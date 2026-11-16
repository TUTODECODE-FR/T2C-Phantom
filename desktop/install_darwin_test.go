//go:build darwin

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallPlanDarwin(t *testing.T) {
	home, _ := os.UserHomeDir()
	exe := func(dir string) string {
		return filepath.Join(dir, "T2C Phantom.app", "Contents", "MacOS", "T2C-Phantom")
	}
	if installPlan(exe("/Applications")) != nil {
		t.Fatal("déjà dans /Applications : rien à proposer")
	}
	if installPlan(exe(filepath.Join(home, "Applications"))) != nil {
		t.Fatal("déjà dans ~/Applications : rien à proposer")
	}
	if installPlan(exe(filepath.Join(home, "Downloads"))) == nil {
		t.Fatal("dans Téléchargements : il faut proposer le déplacement")
	}
	if installPlan("/usr/local/bin/T2C-Phantom") != nil {
		t.Fatal("hors bundle .app : rien à proposer")
	}
}
