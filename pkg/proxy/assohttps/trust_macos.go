package assohttps

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// EnsureTrusted installe la CA dans Firefox (NSS) en arrière-plan.
// Keychain macOS est skippé ici (demande un prompt GUI qui bloquerait le proxy).
func (c *CA) EnsureTrusted() {
	if c == nil {
		return
	}
	go c.ensureTrustedAsync()
}

func (c *CA) ensureTrustedAsync() {
	certPath := c.CertPEMPath()
	if _, err := os.Stat(certPath); err != nil {
		return
	}
	marker := filepath.Join(c.dir, ".trusted-v1")
	if _, err := os.Stat(marker); err == nil {
		return
	}

	// Laisse le proxy binder 8080 avant tout I/O certutil.
	time.Sleep(500 * time.Millisecond)

	ok := trustFirefoxNSS(certPath)
	if ok {
		_ = os.WriteFile(marker, []byte("ok\n"), 0644)
		fmt.Println("✅ CA asso trustée dans Firefox (NSS) — https://*.t2c asso OK")
		return
	}
	fmt.Println("⚠️  CA asso: trust Firefox auto impossible (brew install nss)")
	fmt.Println("   puis: bash infra/install_asso_ca_firefox.sh")
	fmt.Println("   cert:", certPath)
}

func certutilBin() string {
	for _, p := range []string{
		"certutil",
		"/opt/homebrew/opt/nss/bin/certutil",
		"/usr/local/opt/nss/bin/certutil",
	} {
		if p == "certutil" {
			if _, err := exec.LookPath(p); err == nil {
				return p
			}
			continue
		}
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func trustFirefoxNSS(certPath string) bool {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return false
	}
	bin := certutilBin()
	if bin == "" {
		return false
	}
	home, _ := os.UserHomeDir()
	var roots []string
	switch runtime.GOOS {
	case "darwin":
		roots = []string{
			filepath.Join(home, "Library/Application Support/Firefox/Profiles"),
			filepath.Join(home, "Library/Application Support/zen/Profiles"), // Zen Browser
			filepath.Join(home, "Library/Application Support/LibreWolf/Profiles"),
		}
	case "linux":
		roots = []string{
			filepath.Join(home, ".mozilla/firefox"),
			filepath.Join(home, "snap/firefox/common/.mozilla/firefox"),
			filepath.Join(home, ".zen/profiles"),
			filepath.Join(home, ".librewolf"),
		}
	}
	installed := false
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			prof := filepath.Join(root, e.Name())
			for _, prefix := range []string{"sql:", "dbm:"} {
				cmd := exec.Command(bin, "-A", "-n", Nick(), "-t", "CT,C,C", "-i", certPath, "-d", prefix+prof)
				if err := cmd.Run(); err == nil {
					installed = true
					break
				}
			}
		}
	}
	return installed
}
