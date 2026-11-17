package sysint

import "os/exec"

func detectBrowsers() []Browser {
	cands := []struct {
		id, name string
		bins     []string
	}{
		{"firefox", "Firefox", []string{"firefox", "firefox-esr"}},
		{"zen", "Zen", []string{"zen-browser", "zen"}},
		{"librewolf", "LibreWolf", []string{"librewolf"}},
		{"brave", "Brave", []string{"brave-browser", "brave"}},
		{"chrome", "Chromium / Chrome", []string{"chromium", "chromium-browser", "google-chrome"}},
	}
	var out []Browser
	for _, c := range cands {
		for _, bin := range c.bins {
			if p, err := exec.LookPath(bin); err == nil {
				out = append(out, Browser{ID: c.id, Name: c.name, path: p})
				break
			}
		}
	}
	return out
}

func openWith(b Browser, url string) error { return exec.Command(b.path, url).Start() }
func openDefault(url string) error         { return exec.Command("xdg-open", url).Start() }
