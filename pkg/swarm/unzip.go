package swarm

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UnzipDirectory extracts a ZIP byte array into a target directory
func UnzipDirectory(zipBytes []byte, targetDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return err
	}

	os.MkdirAll(targetDir, 0755)

	for _, f := range reader.File {
		// Prevent ZipSlip vulnerability
		if strings.Contains(f.Name, "..") {
			continue
		}

		fpath := filepath.Join(targetDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
