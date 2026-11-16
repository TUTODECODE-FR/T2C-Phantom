//go:build darwin

package sysint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func loginKeychain() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library/Keychains/login.keychain-db")
}

// installCA : trousseau de session (Safari, Chrome, Edge, Brave… et Firefox via
// enterprise_roots) + bases NSS des profils Firefox/Zen si certutil est présent.
func installCA(r *Report, certPath string, der []byte, nick string) {
	fp := fpSHA256(der)
	if !keychainHas(fp) {
		// macOS demande le mot de passe de session (fenêtre système)
		err := exec.Command("security", "add-trusted-cert", "-r", "trustRoot", "-p", "ssl",
			"-k", loginKeychain(), certPath).Run()
		if err != nil {
			r.errf("trousseau macOS : refusé ou annulé (%v)", err)
		}
	}
	if bin := certutilBin(); bin != "" {
		for _, p := range GeckoProfiles() {
			if !nssHas(bin, p, nick, fp) {
				_ = nssInstall(bin, p, certPath, nick)
			}
		}
	}
	r.CA = append(r.CA, caLocations(der, nick)...)
}

func keychainHas(fp string) bool {
	out, err := exec.Command("security", "find-certificate", "-a", "-Z", "-c", "tutodecode CA").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.ReplaceAll(string(out), " ", "")), "sha-256hash:"+fp)
}

func caLocations(der []byte, nick string) []string {
	fp := fpSHA256(der)
	var where []string
	if keychainHas(fp) {
		where = append(where, "Trousseau macOS (Safari, Chrome, Edge, Brave)")
	}
	if bin := certutilBin(); bin != "" {
		for _, p := range GeckoProfiles() {
			if nssHas(bin, p, nick, fp) {
				where = append(where, profileLabel(p))
			}
		}
	}
	return where
}

// services : services réseau actifs (Wi-Fi, Ethernet…).
func services() []string {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil
	}
	var s []string
	for i, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if i == 0 || line == "" || strings.HasPrefix(line, "*") { // en-tête / service désactivé
			continue
		}
		s = append(s, line)
	}
	return s
}

func autoProxy(svc string) (url string, on bool) {
	out, err := exec.Command("networksetup", "-getautoproxyurl", svc).Output()
	if err != nil {
		return "", false
	}
	for _, l := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(l, "URL: "); ok && v != "(null)" {
			url = strings.TrimSpace(v)
		}
		if strings.HasPrefix(l, "Enabled: Yes") {
			on = true
		}
	}
	return
}

func configureProxy(r *Report, dataDir, pacURL string) {
	st := loadState(dataDir)
	for _, svc := range services() {
		url, on := autoProxy(svc)
		if url == pacURL && on {
			r.Proxy = append(r.Proxy, svc)
			continue
		}
		if _, saved := st.Proxies[svc]; !saved {
			prev := ""
			if on {
				prev = url
			}
			st.Proxies[svc] = prev
		}
		if exec.Command("networksetup", "-setautoproxyurl", svc, pacURL).Run() == nil &&
			exec.Command("networksetup", "-setautoproxystate", svc, "on").Run() == nil {
			r.Proxy = append(r.Proxy, svc)
		} else {
			r.errf("proxy %s : modification refusée", svc)
		}
	}
	st.PAC = pacURL
	saveState(dataDir, st)
}

func restoreProxy(dataDir string) error {
	st := loadState(dataDir)
	for svc, prev := range st.Proxies {
		if prev == "" {
			_ = exec.Command("networksetup", "-setautoproxystate", svc, "off").Run()
		} else {
			_ = exec.Command("networksetup", "-setautoproxyurl", svc, prev).Run()
		}
	}
	return os.Remove(statePath(dataDir))
}

func proxyLocations(pacURL string) []string {
	var on []string
	for _, svc := range services() {
		if url, enabled := autoProxy(svc); enabled && url == pacURL {
			on = append(on, svc)
		}
	}
	return on
}
