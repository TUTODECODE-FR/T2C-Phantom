package node

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	maxExtracted = 250 << 20 // 250 Mo décompressés (anti zip bomb)
	maxFiles     = 20000
)

// Unzip extrait un site (déjà vérifié par son manifeste signé) dans dir.
// Défense en profondeur : refuse liens symboliques, chemins sortants, zip bombs.
func Unzip(data []byte, dir string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	if len(zr.File) > maxFiles {
		return fmt.Errorf("trop de fichiers (%d)", len(zr.File))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := filepath.Clean(dir) + string(os.PathSeparator)
	var total int64
	for _, f := range zr.File {
		if f.Mode()&os.ModeSymlink != 0 || strings.Contains(f.Name, `\`) {
			return fmt.Errorf("entrée refusée: %q", f.Name)
		}
		p := filepath.Join(dir, f.Name)
		if !strings.HasPrefix(filepath.Clean(p)+string(os.PathSeparator), base) || filepath.Clean(p) == filepath.Clean(dir) {
			if f.FileInfo().IsDir() && filepath.Clean(p) == filepath.Clean(dir) {
				continue
			}
			return fmt.Errorf("chemin sortant refusé: %q", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(p, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		n, err := io.Copy(out, io.LimitReader(rc, maxExtracted-total+1))
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
		total += n
		if total > maxExtracted {
			return errors.New("site trop gros une fois décompressé (zip bomb ?)")
		}
	}
	return nil
}
