//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const appName = "T2C Phantom.app"

// bundlePath : …/T2C Phantom.app à partir de …/Contents/MacOS/T2C-Phantom.
func bundlePath(exe string) string {
	return filepath.Dir(filepath.Dir(filepath.Dir(exe)))
}

func installPlan(exe string) *installer {
	bundle := bundlePath(exe)
	if !strings.HasSuffix(bundle, ".app") {
		return nil
	}
	home, _ := os.UserHomeDir()
	for _, dir := range []string{"/Applications", filepath.Join(home, "Applications")} {
		if strings.HasPrefix(bundle, dir+"/") {
			return nil // déjà dans Applications
		}
	}
	// Lancée depuis le zip téléchargé : macOS l'exécute depuis une copie
	// temporaire en lecture seule (« App Translocation ») qu'on ne peut pas
	// déplacer ; on la copie, et l'original reste dans Téléchargements.
	translocated := strings.Contains(bundle, "/AppTranslocation/")
	return &installer{
		title:   "Déplacer dans Applications ?",
		message: "T2C Phantom n'est pas dans le dossier Applications. Le déplacer pour le retrouver dans le Launchpad et le Dock ?",
		install: func() (string, error) {
			dst := filepath.Join("/Applications", appName)
			if !writable("/Applications") {
				dst = filepath.Join(home, "Applications", appName)
				_ = os.MkdirAll(filepath.Dir(dst), 0o755)
			}
			if err := os.RemoveAll(dst); err != nil { // ancienne version
				return "", fmt.Errorf("impossible de remplacer %s : %v", dst, err)
			}
			if out, err := exec.Command("ditto", bundle, dst).CombinedOutput(); err != nil {
				return "", fmt.Errorf("copie vers %s : %v %s", dst, err, out)
			}
			if !translocated {
				_ = os.RemoveAll(bundle) // déplacement : on retire l'original
			}
			return dst, nil
		},
	}
}

func writable(dir string) bool {
	f, err := os.CreateTemp(dir, ".t2c-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// relaunch ouvre la nouvelle copie (nouvelle instance) ; l'appelant quitte ensuite.
func relaunch(appPath string) error {
	return startDetached("open", "-n", appPath)
}

func detach(*exec.Cmd) {}
