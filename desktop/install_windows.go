//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const exeName = "T2C-Phantom.exe"

// installDir : %LocalAppData%\Programs\T2C Phantom (aucun droit administrateur requis).
func installDir() string {
	return filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "T2C Phantom")
}

func installPlan(exe string) *installer {
	dir := installDir()
	if strings.EqualFold(filepath.Dir(exe), dir) {
		return nil
	}
	return &installer{
		title: "Installer T2C Phantom ?",
		message: "Installer T2C Phantom sur cet ordinateur, avec un raccourci sur le Bureau et dans le menu Démarrer ?\n\n" +
			"Emplacement : " + dir,
		install: func() (string, error) {
			dst := filepath.Join(dir, exeName)
			if err := copyFile(exe, dst, 0o755); err != nil {
				return "", fmt.Errorf("copie vers %s : %v", dst, err)
			}
			if err := shortcuts(dst); err != nil {
				return "", fmt.Errorf("raccourcis : %v", err)
			}
			return dst, nil
		},
	}
}

// shortcuts crée « T2C Phantom » sur le Bureau et dans le menu Démarrer.
func shortcuts(target string) error {
	ps := `$w = New-Object -ComObject WScript.Shell
foreach ($d in @([Environment]::GetFolderPath('Desktop'), [Environment]::GetFolderPath('Programs'))) {
  $s = $w.CreateShortcut((Join-Path $d 'T2C Phantom.lnk'))
  $s.TargetPath = $env:T2C_TARGET
  $s.WorkingDirectory = Split-Path $env:T2C_TARGET
  $s.IconLocation = $env:T2C_TARGET + ',0'
  $s.Description = 'T2C Phantom'
  $s.Save()
}`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", ps)
	cmd.Env = append(os.Environ(), "T2C_TARGET="+target)
	hide(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v %s", err, out)
	}
	return nil
}

func relaunch(exe string) error { return startDetached(exe) }

const createNoWindow = 0x08000000

func hide(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

func detach(cmd *exec.Cmd) {
	const detachedProcess = 0x00000008
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess}
}
