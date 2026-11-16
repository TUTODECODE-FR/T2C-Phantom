//go:build linux

package sysint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// installCA : Linux n'a pas de magasin utilisateur commun. On remplit les bases
// NSS de Chrome/Chromium (~/.pki/nssdb) et des profils Firefox/Zen avec
// certutil (paquet libnss3-tools / nss-tools) — sans droits administrateur.
func installCA(r *Report, certPath string, der []byte, nick string) {
	bin := certutilBin()
	if bin == "" {
		r.manual("Installe « libnss3-tools » (Debian/Ubuntu) ou « nss-tools » (Fedora), puis relance la configuration automatique")
		return
	}
	fp := fpSHA256(der)
	for _, db := range nssDBs(true) {
		if !nssHas(bin, db, nick, fp) {
			if err := nssInstall(bin, db, certPath, nick); err != nil {
				r.errf("NSS %s : %v", db, err)
			}
		}
	}
	r.CA = append(r.CA, caLocations(der, nick)...)
}

// nssDBs : base Chrome/Chromium (créée si besoin) + profils Gecko.
func nssDBs(create bool) []string {
	home, _ := os.UserHomeDir()
	chrome := filepath.Join(home, ".pki", "nssdb")
	dbs := GeckoProfiles()
	if _, err := os.Stat(filepath.Join(chrome, "cert9.db")); err == nil {
		dbs = append(dbs, chrome)
	} else if create {
		if bin := certutilBin(); bin != "" && os.MkdirAll(chrome, 0o700) == nil &&
			exec.Command(bin, "-N", "--empty-password", "-d", "sql:"+chrome).Run() == nil {
			dbs = append(dbs, chrome)
		}
	}
	return dbs
}

func caLocations(der []byte, nick string) []string {
	bin := certutilBin()
	if bin == "" {
		return nil
	}
	fp := fpSHA256(der)
	var where []string
	for _, db := range nssDBs(false) {
		if nssHas(bin, db, nick, fp) {
			if strings.Contains(db, ".pki") {
				where = append(where, "Chrome / Chromium / Brave")
			} else {
				where = append(where, profileLabel(db))
			}
		}
	}
	return where
}

// GNOME (et dérivés) : proxy automatique par PAC. KDE et autres : manuel.
func gsettings(args ...string) (string, error) {
	out, err := exec.Command("gsettings", args...).Output()
	return strings.Trim(strings.TrimSpace(string(out)), "'"), err
}

func configureProxy(r *Report, dataDir, pacURL string) {
	if _, err := exec.LookPath("gsettings"); err != nil {
		r.manual("Proxy : configure l'URL de configuration automatique %s dans les réglages réseau", pacURL)
		return
	}
	st := loadState(dataDir)
	mode, _ := gsettings("get", "org.gnome.system.proxy", "mode")
	url, _ := gsettings("get", "org.gnome.system.proxy", "autoconfig-url")
	if _, saved := st.Proxies["mode"]; !saved && !(mode == "auto" && url == pacURL) {
		st.Proxies["mode"], st.Proxies["autoconfig-url"] = mode, url
	}
	if exec.Command("gsettings", "set", "org.gnome.system.proxy", "autoconfig-url", pacURL).Run() != nil ||
		exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", "auto").Run() != nil {
		r.manual("Proxy : configure l'URL de configuration automatique %s dans les réglages réseau", pacURL)
		return
	}
	st.PAC = pacURL
	saveState(dataDir, st)
	r.Proxy = append(r.Proxy, "GNOME (Chrome, Firefox)")
}

func restoreProxy(dataDir string) error {
	st := loadState(dataDir)
	if mode, ok := st.Proxies["mode"]; ok {
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "autoconfig-url", st.Proxies["autoconfig-url"]).Run()
		_ = exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", mode).Run()
	}
	return os.Remove(statePath(dataDir))
}

func proxyLocations(pacURL string) []string {
	mode, err := gsettings("get", "org.gnome.system.proxy", "mode")
	if err != nil {
		return nil
	}
	if url, _ := gsettings("get", "org.gnome.system.proxy", "autoconfig-url"); mode == "auto" && url == pacURL {
		return []string{"GNOME (Chrome, Firefox)"}
	}
	return nil
}
