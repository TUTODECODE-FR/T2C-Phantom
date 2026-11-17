package sysint

import (
	"os"
	"os/exec"
	"path/filepath"
)

func detectBrowsers() []Browser {
	pf, pf86, local := os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LocalAppData")
	cands := []struct {
		id, name string
		paths    []string
	}{
		{"firefox", "Firefox", []string{filepath.Join(pf, `Mozilla Firefox\firefox.exe`), filepath.Join(pf86, `Mozilla Firefox\firefox.exe`)}},
		{"zen", "Zen", []string{filepath.Join(pf, `Zen Browser\zen.exe`), filepath.Join(local, `Zen Browser\zen.exe`)}},
		{"librewolf", "LibreWolf", []string{filepath.Join(pf, `LibreWolf\librewolf.exe`)}},
		{"brave", "Brave", []string{filepath.Join(pf, `BraveSoftware\Brave-Browser\Application\brave.exe`), filepath.Join(local, `BraveSoftware\Brave-Browser\Application\brave.exe`)}},
		{"chrome", "Chrome", []string{filepath.Join(pf, `Google\Chrome\Application\chrome.exe`), filepath.Join(pf86, `Google\Chrome\Application\chrome.exe`), filepath.Join(local, `Google\Chrome\Application\chrome.exe`)}},
		{"edge", "Edge", []string{filepath.Join(pf86, `Microsoft\Edge\Application\msedge.exe`), filepath.Join(pf, `Microsoft\Edge\Application\msedge.exe`)}},
	}
	var out []Browser
	for _, c := range cands {
		for _, p := range c.paths {
			if _, err := os.Stat(p); err == nil {
				out = append(out, Browser{ID: c.id, Name: c.name, path: p})
				break
			}
		}
	}
	return out
}

func openWith(b Browser, url string) error { return exec.Command(b.path, url).Start() }
func openDefault(url string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
}
