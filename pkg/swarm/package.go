package swarm

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipDirectory compresses a directory into a byte array
func ZipDirectory(source string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		
		// Always use forward slashes for ZIP paths
		relPath = strings.ReplaceAll(relPath, "\\", "/")

		f, err := w.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(f, file)
		return err
	})

	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

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
