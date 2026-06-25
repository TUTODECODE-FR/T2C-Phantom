package swarm

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"t2c/pkg/utils"
)

// UnzipDirectory extracts a ZIP byte array into a target directory
func UnzipDirectory(zipBytes []byte, targetDir string) error {
	reader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return err
	}

	os.MkdirAll(targetDir, 0755)

	var totalExtracted int64
	maxSize := int64(250 * 1024 * 1024) // 250 MB

	for _, f := range reader.File {
		// Prevent Symlink attacks
		if f.Mode()&os.ModeSymlink != 0 {
			utils.LogSecurityEvent("symlink_bypass_attempt", fmt.Sprintf("Blocked extraction of symlink: %s", f.Name), "unknown")
			continue
		}

		fpath := filepath.Join(targetDir, f.Name)
		
		// Prevent ZipSlip vulnerability properly
		if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(targetDir)+string(os.PathSeparator)) {
			utils.LogSecurityEvent("zipslip_attempt", fmt.Sprintf("Blocked ZipSlip: %s", f.Name), "unknown")
			continue
		}

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

		limitReader := io.LimitReader(rc, maxSize-totalExtracted+1)
		n, err := io.Copy(outFile, limitReader)
		totalExtracted += n
		
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
		if totalExtracted > maxSize {
			os.RemoveAll(targetDir)
			utils.LogSecurityEvent("zip_bomb_detected", fmt.Sprintf("Extracted size exceeded limit of %d bytes", maxSize), "unknown")
			return fmt.Errorf("zip bomb detected: exceeded 250MB limit")
		}
	}
	return nil
}
