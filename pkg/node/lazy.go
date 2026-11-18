package node

// Chargement à la demande : un site est une archive zip découpée en morceaux
// signés. Le zip permet de lire un seul fichier sans tout télécharger : on ne
// va chercher (chez les bénévoles, sinon à l'origine) que les morceaux qui
// contiennent la page demandée, chacun vérifié contre le manifeste signé.
// Un gros site (dépôts de code…) s'ouvre ainsi aussi vite qu'un petit.

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"t2c/pkg/peerclient"
	"t2c/pkg/trust"
	"t2c/pkg/wire"
)

// MaxServedFile : taille maximale d'un fichier servi (décompressé).
const MaxServedFile = 96 << 20

type lazySite struct {
	n     *Node
	m     *trust.Manifest
	parts string // cache des morceaux vérifiés

	mu      sync.Mutex
	hosts   []wire.HostInfo
	hostsAt time.Time

	zr    *zip.Reader
	files map[string]*zip.File
	dirs  map[string][]string // dossier → noms directs

	hot   map[int][]byte // derniers morceaux lus (évite de relire le disque)
	hotMu sync.Mutex
}

// readAhead : morceaux suivants demandés en même temps (gros fichiers plus rapides).
const readAhead = 8

// SiteFS : le site sous forme de système de fichiers. Site déjà présent en
// entier (copie de bénévole ou ancienne extraction) : lu localement ; sinon
// chaque fichier est chargé à la demande.
func (n *Node) SiteFS(ctx context.Context, domain string) (http.FileSystem, *trust.SiteEntry, error) {
	e, err := n.Resolve(ctx, domain)
	if err != nil {
		return nil, nil, err
	}
	dir := n.siteDir(e.Root)
	if _, err := os.Stat(filepath.Join(dir, ".t2c-ok")); err == nil {
		return http.Dir(dir), e, nil
	}
	n.mu.Lock()
	ls := n.lazy[e.Root]
	n.mu.Unlock()
	if ls != nil {
		return ls, e, nil
	}
	c, err := n.waitClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	m, err := c.Manifest(ctx, e.Root)
	if err != nil {
		return nil, nil, err
	}
	ls = &lazySite{n: n, m: m, parts: dir + ".parts"}
	if err := os.MkdirAll(ls.parts, 0o700); err != nil {
		return nil, nil, err
	}
	var ra io.ReaderAt = ls
	if data, _ := n.store.Read(e.Root); data != nil { // copie hébergée (revérifiée à la lecture)
		ra = bytes.NewReader(data)
	} else if n.hostingPrefs().wants(e.ID) {
		go n.hostInBackground(c, e) // bénévole : copie complète en arrière-plan, la page s'affiche déjà
	}
	zr, err := zip.NewReader(ra, m.Size)
	if err != nil {
		return nil, nil, fmt.Errorf("site illisible : %w", err)
	}
	ls.index(zr)
	n.mu.Lock()
	if prev := n.lazy[e.Root]; prev != nil {
		ls = prev
	} else {
		n.lazy[e.Root] = ls
	}
	n.mu.Unlock()
	go n.notePeers(context.Background(), c, e.Root)
	return ls, e, nil
}

func (n *Node) hostInBackground(c *peerclient.Client, e *trust.SiteEntry) {
	n.mu.Lock()
	lk := n.fetching[e.Root]
	if lk == nil {
		lk = &sync.Mutex{}
		n.fetching[e.Root] = lk
	}
	n.mu.Unlock()
	if !lk.TryLock() {
		return // déjà en cours
	}
	defer lk.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if _, err := c.Host(ctx, e.ID); err != nil {
		n.setErr(fmt.Errorf("bénévolat %s: %w", e.ID, err))
		return
	}
	n.saveTickets(c.Tickets())
}

func (ls *lazySite) index(zr *zip.Reader) {
	ls.zr = zr
	ls.files = map[string]*zip.File{}
	ls.dirs = map[string][]string{"": nil}
	seen := map[string]bool{}
	for _, f := range zr.File {
		name := strings.TrimPrefix(path.Clean("/"+f.Name), "/")
		if name == "" || strings.HasSuffix(f.Name, "/") || f.Mode()&fs.ModeType != 0 {
			continue // dossiers implicites ; liens symboliques et fichiers spéciaux jamais servis
		}
		ls.files[name] = f
		// rattache chaque élément du chemin à son dossier parent
		for child := name; child != ""; {
			parent := path.Dir(child)
			if parent == "." {
				parent = ""
			}
			if !seen[child] {
				seen[child] = true
				ls.dirs[parent] = append(ls.dirs[parent], path.Base(child))
			}
			child = parent
		}
	}
}

// ReadAt : octets [off, off+len(p)) du site, en ne chargeant que les morceaux nécessaires.
func (ls *lazySite) ReadAt(p []byte, off int64) (int, error) {
	if off >= ls.m.Size {
		return 0, io.EOF
	}
	end := off + int64(len(p))
	short := false
	if end > ls.m.Size {
		end, short = ls.m.Size, true
	}
	cs := int64(ls.m.ChunkSize)
	first, last := int(off/cs), int((end-1)/cs)
	chunks, err := ls.chunks(first, last)
	if err != nil {
		return 0, err
	}
	n := 0
	for i := first; i <= last; i++ {
		start := int64(i) * cs
		lo, hi := max(off, start)-start, min(end, start+int64(len(chunks[i])))-start
		n += copy(p[n:], chunks[i][lo:hi])
	}
	if short {
		return n, io.EOF
	}
	return n, nil
}

func (ls *lazySite) partPath(i int) string { return filepath.Join(ls.parts, strconv.Itoa(i)) }

// chunks : morceaux first..last, depuis le cache (revérifiés) ou le réseau.
func (ls *lazySite) chunks(first, last int) (map[int][]byte, error) {
	out := map[int][]byte{}
	var missing []int
	ls.hotMu.Lock()
	for i := first; i <= last; i++ {
		if b, ok := ls.hot[i]; ok {
			out[i] = b
		}
	}
	ls.hotMu.Unlock()
	for i := first; i <= last; i++ {
		if out[i] != nil {
			continue
		}
		if b, err := os.ReadFile(ls.partPath(i)); err == nil && ls.m.VerifyChunk(i, b) {
			out[i] = b
		} else {
			missing = append(missing, i)
		}
	}
	if len(missing) == 0 {
		ls.remember(out)
		return out, nil
	}
	// lecture anticipée : un fichier se lit d'un bout à l'autre
	for i := last + 1; i < len(ls.m.Chunks) && i <= last+readAhead; i++ {
		if _, err := os.Stat(ls.partPath(i)); err != nil {
			missing = append(missing, i)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, err := ls.n.waitClient(ctx)
	if err != nil {
		return nil, err
	}
	st := &peerclient.FetchStats{}
	got, err := c.FetchChunks(ctx, ls.m, missing, ls.peers(ctx, c), st)
	ls.n.addStats(st)
	if err != nil {
		return nil, err
	}
	for i, b := range got {
		_ = os.WriteFile(ls.partPath(i), b, 0o600)
		if i >= first && i <= last {
			out[i] = b
		}
	}
	ls.remember(out)
	return out, nil
}

func (ls *lazySite) remember(m map[int][]byte) {
	ls.hotMu.Lock()
	defer ls.hotMu.Unlock()
	if ls.hot == nil || len(ls.hot) > 16 {
		ls.hot = map[int][]byte{}
	}
	for i, b := range m {
		ls.hot[i] = b
	}
}

// Stream : un gros fichier lu au fil de l'eau (le navigateur affiche la
// progression) au lieu d'être assemblé en mémoire avant d'être envoyé.
func (ls *lazySite) Stream(name string) (io.ReadCloser, int64, bool) {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	f, ok := ls.files[name]
	if !ok || f.UncompressedSize64 > MaxServedFile || f.UncompressedSize64 < StreamFrom {
		return nil, 0, false
	}
	rc, err := f.Open()
	if err != nil {
		return nil, 0, false
	}
	return rc, int64(f.UncompressedSize64), true
}

// StreamFrom : taille à partir de laquelle un fichier est envoyé au fil de l'eau.
const StreamFrom = 1 << 20

// peers : bénévoles du site, redemandés au plus toutes les 2 minutes.
func (ls *lazySite) peers(ctx context.Context, c *peerclient.Client) []wire.HostInfo {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if time.Since(ls.hostsAt) > 2*time.Minute {
		ls.hosts, ls.hostsAt = c.Peers(ctx, ls.m.Root), time.Now()
	}
	return ls.hosts
}

// --- http.FileSystem -------------------------------------------------------------

func (ls *lazySite) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(path.Clean("/"+name), "/")
	if f, ok := ls.files[name]; ok {
		if f.UncompressedSize64 > MaxServedFile {
			return nil, fs.ErrPermission
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer rc.Close()
		data, err := io.ReadAll(io.LimitReader(rc, MaxServedFile+1))
		if err != nil {
			return nil, err // morceau introuvable, ou contenu ne correspondant pas à sa somme
		}
		if len(data) > MaxServedFile {
			return nil, fs.ErrPermission
		}
		return &memFile{Reader: bytes.NewReader(data), info: fileInfo{name: path.Base(name), size: int64(len(data)), mod: f.Modified}}, nil
	}
	if kids, ok := ls.dirs[name]; ok {
		return &dirFile{ls: ls, name: name, kids: kids}, nil
	}
	return nil, fs.ErrNotExist
}

type fileInfo struct {
	name string
	size int64
	mod  time.Time
	dir  bool
}

func (i fileInfo) Name() string { return i.name }
func (i fileInfo) Size() int64  { return i.size }
func (i fileInfo) Mode() fs.FileMode {
	if i.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}
func (i fileInfo) ModTime() time.Time { return i.mod }
func (i fileInfo) IsDir() bool        { return i.dir }
func (i fileInfo) Sys() any           { return nil }

type memFile struct {
	*bytes.Reader
	info fileInfo
}

func (f *memFile) Close() error                       { return nil }
func (f *memFile) Readdir(int) ([]fs.FileInfo, error) { return nil, errors.New("pas un dossier") }
func (f *memFile) Stat() (fs.FileInfo, error)         { return f.info, nil }

type dirFile struct {
	ls   *lazySite
	name string
	kids []string
}

func (d *dirFile) Close() error                   { return nil }
func (d *dirFile) Read([]byte) (int, error)       { return 0, errors.New("dossier") }
func (d *dirFile) Seek(int64, int) (int64, error) { return 0, nil }
func (d *dirFile) Stat() (fs.FileInfo, error) {
	return fileInfo{name: path.Base("/" + d.name), dir: true}, nil
}
func (d *dirFile) Readdir(int) ([]fs.FileInfo, error) {
	var out []fs.FileInfo
	kids := append([]string(nil), d.kids...)
	sort.Strings(kids)
	for _, k := range kids {
		full := strings.TrimPrefix(d.name+"/"+k, "/")
		if f, ok := d.ls.files[full]; ok {
			out = append(out, fileInfo{name: k, size: int64(f.UncompressedSize64), mod: f.Modified})
		} else {
			out = append(out, fileInfo{name: k, dir: true})
		}
	}
	return out, nil
}
