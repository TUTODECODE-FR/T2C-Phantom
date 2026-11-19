package node

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Archives de site reçues de bénévoles : rien ne doit sortir du dossier cible
// (chemins « ../ », absolus, liens), quelle que soit l'archive.
func FuzzUnzip(f *testing.F) {
	mk := func(names ...string) []byte {
		var b bytes.Buffer
		w := zip.NewWriter(&b)
		for _, n := range names {
			fw, _ := w.Create(n)
			_, _ = fw.Write([]byte("x"))
		}
		_ = w.Close()
		return b.Bytes()
	}
	f.Add(mk("index.html", "a/b.css"))
	f.Add(mk("../evil", "/abs", "a/../../x", `..\win`))
	f.Add([]byte("PK\x03\x04"))
	f.Fuzz(func(t *testing.T, data []byte) {
		root := t.TempDir()
		dir := filepath.Join(root, "site")
		_ = Unzip(data, dir)
		es, err := os.ReadDir(root)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range es {
			if e.Name() != "site" {
				t.Fatalf("fichier écrit hors du dossier : %s", e.Name())
			}
		}
		_ = filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err == nil && d.Type()&os.ModeSymlink != 0 {
				t.Fatalf("lien symbolique créé : %s", p)
			}
			return nil
		})
	})
}
