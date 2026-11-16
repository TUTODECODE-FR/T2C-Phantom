package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"t2c/pkg/utils"
)

// Au premier lancement hors de l'emplacement standard, l'app propose de
// s'installer (Applications sur macOS, Programmes + raccourcis sur Windows,
// ~/.local + menu sur Linux), puis se relance depuis là.

var installOnce sync.Once

func skipFile() string { return filepath.Join(utils.GetT2CDataDir(), "install-prompt-skip") }

// offerInstall : appelée quand l'interface est prête.
func offerInstall(ctx context.Context) {
	installOnce.Do(func() {
		if os.Getenv("T2C_NO_INSTALL_PROMPT") != "" || version == "dev" {
			return // build de développement : on ne déplace rien
		}
		if _, err := os.Stat(skipFile()); err == nil {
			return
		}
		exe, err := os.Executable()
		if err != nil {
			return
		}
		exe, _ = filepath.EvalSymlinks(exe)
		plan := installPlan(exe)
		if plan == nil {
			return // déjà installée
		}
		choice := ask(ctx, plan.title, plan.message)
		switch choice {
		case "install":
			newExe, err := plan.install()
			if err != nil {
				runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
					Type: runtime.ErrorDialog, Title: "Installation impossible",
					Message: err.Error(),
				})
				return
			}
			if relaunch(newExe) == nil {
				runtime.Quit(ctx)
			}
		case "never":
			_ = os.MkdirAll(filepath.Dir(skipFile()), 0o700)
			_ = os.WriteFile(skipFile(), []byte("1\n"), 0o600)
		}
	})
}

type installer struct {
	title, message string
	install        func() (newExe string, err error)
}

// ask : macOS accepte des boutons personnalisés ; Windows/Linux seulement Oui/Non.
func ask(ctx context.Context, title, msg string) string {
	const yes, later, never = "Installer", "Plus tard", "Ne plus demander"
	res, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{
		Type: runtime.QuestionDialog, Title: title, Message: msg,
		Buttons: []string{yes, later, never}, DefaultButton: yes, CancelButton: later,
	})
	if err != nil {
		return "later"
	}
	switch strings.ToLower(res) {
	case strings.ToLower(yes), "yes", "oui", "ok":
		return "install"
	case strings.ToLower(never):
		return "never"
	}
	return "later"
}

// copyFile copie un fichier en conservant ses droits d'exécution.
func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".new"
	if err := os.WriteFile(tmp, data, mode); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func startDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	detach(cmd)
	return cmd.Start()
}
