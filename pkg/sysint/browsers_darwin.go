package sysint

import (
	"os"
	"os/exec"
	"path/filepath"
)

var macBrowsers = []struct{ id, name, app string }{
	{"firefox", "Firefox", "Firefox.app"},
	{"zen", "Zen", "Zen.app"},
	{"zen", "Zen", "Zen Browser.app"},
	{"librewolf", "LibreWolf", "LibreWolf.app"},
	{"brave", "Brave", "Brave Browser.app"},
	{"chrome", "Chrome", "Google Chrome.app"},
	{"edge", "Edge", "Microsoft Edge.app"},
	{"safari", "Safari", "Safari.app"},
}

func detectBrowsers() []Browser {
	home, _ := os.UserHomeDir()
	seen := map[string]bool{}
	var out []Browser
	for _, b := range macBrowsers {
		if seen[b.id] {
			continue
		}
		for _, dir := range []string{"/Applications", filepath.Join(home, "Applications"), "/System/Applications", "/System/Cryptexes/App/System/Applications"} {
			p := filepath.Join(dir, b.app)
			if _, err := os.Stat(p); err == nil {
				out = append(out, Browser{ID: b.id, Name: b.name, path: p})
				seen[b.id] = true
				break
			}
		}
	}
	return out
}

func openWith(b Browser, url string) error { return exec.Command("open", "-a", b.path, url).Start() }
func openDefault(url string) error         { return exec.Command("open", url).Start() }
