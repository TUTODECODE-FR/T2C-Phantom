package node

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkzip(t *testing.T, entries map[string]string, mode map[string]os.FileMode) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	for name, body := range entries {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		if m, ok := mode[name]; ok {
			h.SetMode(m)
		}
		f, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.Write([]byte(body))
	}
	_ = w.Close()
	return b.Bytes()
}

// Archives piégées : rien ne doit être écrit hors du dossier du site.
func TestUnzipAttacks(t *testing.T) {
	cases := map[string][]byte{
		"remontée ../":           mkzip(t, map[string]string{"../evil.txt": "x"}, nil),
		"remontée profonde":      mkzip(t, map[string]string{"a/../../../evil.txt": "x"}, nil),
		"chemin absolu":          mkzip(t, map[string]string{"/tmp/t2c-evil.txt": "x"}, nil),
		"antislash Windows":      mkzip(t, map[string]string{`..\evil.txt`: "x"}, nil),
		"lien symbolique":        mkzip(t, map[string]string{"lien": "/etc/passwd"}, map[string]os.FileMode{"lien": os.ModeSymlink | 0o777}),
		"bombe de décompression": mkzip(t, map[string]string{"bombe.bin": strings.Repeat("\x00", 600<<20)}, nil),
	}
	for name, data := range cases {
		parent := t.TempDir()
		dir := filepath.Join(parent, "site")
		err := Unzip(data, dir)
		var outside []string
		_ = filepath.Walk(parent, func(p string, info os.FileInfo, _ error) error {
			if info != nil && !info.IsDir() && !strings.HasPrefix(p, dir+string(os.PathSeparator)) {
				outside = append(outside, p)
			}
			return nil
		})
		if _, e := os.Stat("/tmp/t2c-evil.txt"); e == nil {
			outside = append(outside, "/tmp/t2c-evil.txt")
			_ = os.Remove("/tmp/t2c-evil.txt")
		}
		if len(outside) > 0 {
			t.Fatalf("%s : fichier écrit hors du site : %v", name, outside)
		}
		t.Logf("%s → %v", name, err)
		if name == "bombe de décompression" && err == nil {
			t.Fatalf("bombe acceptée")
		}
	}
}
