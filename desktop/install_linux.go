//go:build linux

package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

//go:embed build/appicon.png
var appIcon []byte

func installPlan(exe string) *installer {
	home, _ := os.UserHomeDir()
	dst := filepath.Join(home, ".local", "bin", "t2c-phantom")
	if exe == dst {
		return nil
	}
	return &installer{
		title: "Installer T2C Phantom ?",
		message: "Installer T2C Phantom pour ton compte, avec une entrée dans le menu des applications et un raccourci sur le Bureau ?\n\n" +
			"Emplacement : " + dst,
		install: func() (string, error) {
			if err := copyFile(exe, dst, 0o755); err != nil {
				return "", fmt.Errorf("copie vers %s : %v", dst, err)
			}
			icon := filepath.Join(home, ".local", "share", "icons", "hicolor", "512x512", "apps", "t2c-phantom.png")
			if err := os.MkdirAll(filepath.Dir(icon), 0o755); err == nil {
				_ = os.WriteFile(icon, appIcon, 0o644)
			}
			entry := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=T2C Phantom
Comment=Réseau de pairs pseudonymes — TUTODECODE
Exec=%s
Icon=t2c-phantom
Terminal=false
Categories=Network;Education;
`, dst)
			apps := filepath.Join(home, ".local", "share", "applications", "t2c-phantom.desktop")
			if err := os.MkdirAll(filepath.Dir(apps), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(apps, []byte(entry), 0o644); err != nil {
				return "", err
			}
			if desk := desktopDir(home); desk != "" {
				link := filepath.Join(desk, "t2c-phantom.desktop")
				if os.WriteFile(link, []byte(entry), 0o755) == nil {
					// GNOME : marque le lanceur comme fiable (sinon « lancement non autorisé »)
					_ = exec.Command("gio", "set", link, "metadata::trusted", "true").Run()
				}
			}
			_ = exec.Command("update-desktop-database", filepath.Dir(apps)).Run()
			return dst, nil
		},
	}
}

// desktopDir : dossier Bureau (XDG, localisé : « Bureau », « Desktop »…).
func desktopDir(home string) string {
	if out, err := exec.Command("xdg-user-dir", "DESKTOP").Output(); err == nil {
		if d := strings.TrimSpace(string(out)); d != "" && d != home {
			return d
		}
	}
	for _, d := range []string{"Bureau", "Desktop"} {
		if st, err := os.Stat(filepath.Join(home, d)); err == nil && st.IsDir() {
			return filepath.Join(home, d)
		}
	}
	return ""
}

func relaunch(exe string) error { return startDetached(exe) }

func detach(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} }
