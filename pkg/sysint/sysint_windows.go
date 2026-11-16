//go:build windows

package sysint

import (
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const inetKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

// installCA : magasin « Racines de confiance » de l'utilisateur (Edge, Chrome,
// Brave… et Firefox via enterprise_roots). Windows affiche un avertissement
// de sécurité à confirmer. Pas besoin de droits administrateur.
func installCA(r *Report, certPath string, der []byte, nick string) {
	if !storeHas(der) {
		if err := exec.Command("certutil.exe", "-user", "-f", "-addstore", "Root", certPath).Run(); err != nil {
			r.errf("magasin Windows : refusé ou annulé (%v)", err)
		}
	}
	r.CA = append(r.CA, caLocations(der, nick)...)
}

func storeHas(der []byte) bool {
	out, err := exec.Command("certutil.exe", "-user", "-store", "Root", fpSHA1(der)).Output()
	return err == nil && strings.Contains(strings.ToUpper(strings.ReplaceAll(string(out), " ", "")), fpSHA1(der))
}

func caLocations(der []byte, nick string) []string {
	if storeHas(der) {
		return []string{"Magasin Windows (Edge, Chrome, Brave, Firefox)"}
	}
	return nil
}

// notify : prévient WinINet (Edge, Chrome) que les réglages proxy ont changé.
func notify() {
	wininet := syscall.NewLazyDLL("wininet.dll")
	set := wininet.NewProc("InternetSetOptionW")
	const settingsChanged, refresh = 39, 37
	set.Call(0, settingsChanged, 0, 0)
	set.Call(0, refresh, 0, 0)
}

func configureProxy(r *Report, dataDir, pacURL string) {
	k, err := registry.OpenKey(registry.CURRENT_USER, inetKey, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		r.errf("proxy Windows : %v", err)
		return
	}
	defer k.Close()
	st := loadState(dataDir)
	cur, _, _ := k.GetStringValue("AutoConfigURL")
	if _, saved := st.Proxies["AutoConfigURL"]; !saved && cur != pacURL {
		st.Proxies["AutoConfigURL"] = cur
	}
	if err := k.SetStringValue("AutoConfigURL", pacURL); err != nil {
		r.errf("proxy Windows : %v", err)
		return
	}
	notify()
	st.PAC = pacURL
	saveState(dataDir, st)
	r.Proxy = append(r.Proxy, "Windows (Edge, Chrome, Firefox)")
}

func restoreProxy(dataDir string) error {
	st := loadState(dataDir)
	k, err := registry.OpenKey(registry.CURRENT_USER, inetKey, registry.SET_VALUE)
	if err == nil {
		if prev, ok := st.Proxies["AutoConfigURL"]; ok && prev != "" {
			_ = k.SetStringValue("AutoConfigURL", prev)
		} else {
			_ = k.DeleteValue("AutoConfigURL")
		}
		k.Close()
		notify()
	}
	return os.Remove(statePath(dataDir))
}

func proxyLocations(pacURL string) []string {
	k, err := registry.OpenKey(registry.CURRENT_USER, inetKey, registry.QUERY_VALUE)
	if err != nil {
		return nil
	}
	defer k.Close()
	if v, _, _ := k.GetStringValue("AutoConfigURL"); v == pacURL {
		return []string{"Windows (Edge, Chrome, Firefox)"}
	}
	return nil
}
