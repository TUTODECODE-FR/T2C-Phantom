// Package sysint : intégration « plug-and-play » au système.
//
//   - CA locale (.t2c uniquement, Name Constraints) installée dans le magasin
//     du système (Chrome, Edge, Safari…) et dans les profils Firefox / Zen /
//     LibreWolf ;
//   - proxy configuré automatiquement par un fichier PAC qui n'envoie QUE les
//     *.t2c au proxy local ; tout le reste reste en connexion directe.
//
// Les réglages proxy d'origine sont sauvegardés et restaurés à la fermeture.
// Chaque OS a son implémentation (sysint_darwin.go, _windows.go, _linux.go).
package sysint

import (
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Report : ce qui a été fait, ce qui reste manuel.
type Report struct {
	CA     []string `json:"ca"`     // où la CA est de confiance
	Proxy  []string `json:"proxy"`  // où le PAC est actif
	Manual []string `json:"manual"` // étapes restantes pour l'utilisateur
	Errors []string `json:"errors"`
}

func (r *Report) errf(f string, a ...any)   { r.Errors = append(r.Errors, fmt.Sprintf(f, a...)) }
func (r *Report) manual(f string, a ...any) { r.Manual = append(r.Manual, fmt.Sprintf(f, a...)) }

// PAC : seuls les *.t2c passent par le proxy local.
func PAC(proxyAddr string) string {
	return `function FindProxyForURL(url, host) {
  if (dnsDomainIs(host, ".t2c")) return "PROXY ` + proxyAddr + `";
  return "DIRECT";
}
`
}

// PACURL : adresse du PAC servi par le proxy local.
func PACURL(proxyAddr string) string { return "http://" + proxyAddr + "/proxy.pac" }

// --- état sauvegardé (pour restaurer les réglages d'origine) -------------------

type savedState struct {
	PAC     string            `json:"pac"`
	Proxies map[string]string `json:"proxies"` // service/clé → ancienne valeur
}

func statePath(dataDir string) string { return filepath.Join(dataDir, "sysint-state.json") }

func loadState(dataDir string) *savedState {
	st := &savedState{Proxies: map[string]string{}}
	if b, err := os.ReadFile(statePath(dataDir)); err == nil {
		_ = json.Unmarshal(b, st)
	}
	if st.Proxies == nil {
		st.Proxies = map[string]string{}
	}
	return st
}

func saveState(dataDir string, st *savedState) {
	b, _ := json.MarshalIndent(st, "", "  ")
	_ = os.WriteFile(statePath(dataDir), b, 0o600)
}

// --- certificat ------------------------------------------------------------------

func certDER(certPath string) ([]byte, error) {
	b, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, fmt.Errorf("certificat illisible")
	}
	return blk.Bytes, nil
}

func fpSHA256(der []byte) string { s := sha256.Sum256(der); return hex.EncodeToString(s[:]) }
func fpSHA1(der []byte) string   { s := sha1.Sum(der); return strings.ToUpper(hex.EncodeToString(s[:])) }

// --- navigateurs Gecko (Firefox, Zen, LibreWolf) ---------------------------------

func geckoRoots() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return []string{
			filepath.Join(home, "Library/Application Support/Firefox/Profiles"),
			filepath.Join(home, "Library/Application Support/zen/Profiles"),
			filepath.Join(home, "Library/Application Support/LibreWolf/Profiles"),
		}
	case "windows":
		app := os.Getenv("APPDATA")
		return []string{
			filepath.Join(app, "Mozilla", "Firefox", "Profiles"),
			filepath.Join(app, "zen", "Profiles"),
			filepath.Join(app, "librewolf", "Profiles"),
		}
	default:
		return []string{
			filepath.Join(home, ".mozilla/firefox"),
			filepath.Join(home, "snap/firefox/common/.mozilla/firefox"),
			filepath.Join(home, ".var/app/org.mozilla.firefox/.mozilla/firefox"),
			filepath.Join(home, ".zen"),
			filepath.Join(home, ".librewolf"),
		}
	}
}

// GeckoProfiles : profils contenant une base de certificats NSS.
func GeckoProfiles() []string {
	var out []string
	for _, root := range geckoRoots() {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			p := filepath.Join(root, e.Name())
			if _, err := os.Stat(filepath.Join(p, "cert9.db")); err == nil {
				out = append(out, p)
			}
		}
	}
	return out
}

func profileLabel(p string) string {
	app := "Firefox"
	l := strings.ToLower(p)
	switch {
	case strings.Contains(l, "zen"):
		app = "Zen"
	case strings.Contains(l, "librewolf"):
		app = "LibreWolf"
	}
	name := filepath.Base(p)
	if i := strings.IndexByte(name, '.'); i >= 0 && i+1 < len(name) {
		name = name[i+1:]
	}
	return app + " · " + name
}

const userJSBegin = "// >>> T2C Phantom (géré automatiquement)"
const userJSEnd = "// <<< T2C Phantom"

// geckoPrefs : fait lire aux navigateurs Gecko les CA du système, SEULEMENT
// quand certutil manque (sinon la CA est ajoutée à leur base NSS, sans
// toucher aux réglages) et jamais dans LibreWolf, dont les réglages durcis
// sont un choix du membre. Le réglage s'applique à toutes les CA du système :
// il n'est posé qu'à l'étape « certificat » de l'assistant, que le membre
// lance lui-même (accord explicite), et retiré à la désactivation.
// Pour le proxy, Firefox suit par défaut les réglages du système, donc notre PAC.
//
// Aucun réglage WebRTC n'est écrit : couper media.peerconnection.enabled dans
// les profils habituels du membre casserait ses visioconférences. Le risque
// (un script de site .t2c peut obtenir l'IP par STUN) est documenté.
func geckoPrefs(enable bool) []string {
	var done []string
	block := userJSBegin + "\n" +
		`user_pref("security.enterprise_roots.enabled", true);` + "\n" +
		userJSEnd + "\n"
	haveCertutil := certutilBin() != ""
	for _, p := range GeckoProfiles() {
		f := filepath.Join(p, "user.js")
		old, _ := os.ReadFile(f)
		s := stripBlock(string(old))
		want := enable && geckoNeedsPref(p, haveCertutil)
		if want {
			s += block
		}
		if s != string(old) && os.WriteFile(f, []byte(s), 0o644) != nil {
			continue
		}
		if want {
			done = append(done, profileLabel(p))
		}
	}
	return done
}

// geckoNeedsPref : le profil a besoin de lire les CA du système.
func geckoNeedsPref(profile string, haveCertutil bool) bool {
	return !haveCertutil && !strings.Contains(strings.ToLower(profile), "librewolf")
}

func stripBlock(s string) string {
	for {
		a := strings.Index(s, userJSBegin)
		if a < 0 {
			return s
		}
		b := strings.Index(s[a:], userJSEnd)
		if b < 0 {
			return s[:a]
		}
		end := a + b + len(userJSEnd)
		if end < len(s) && s[end] == '\n' {
			end++
		}
		s = s[:a] + s[end:]
	}
}

// --- certutil (NSS) : Firefox / Zen, et Chrome sous Linux -------------------------

func certutilBin() string {
	for _, p := range []string{"certutil", "/opt/homebrew/opt/nss/bin/certutil", "/usr/local/opt/nss/bin/certutil", "/opt/homebrew/bin/certutil"} {
		if runtime.GOOS == "windows" {
			break // certutil.exe de Windows n'est PAS celui de NSS
		}
		if path, err := exec.LookPath(p); err == nil {
			return path
		}
	}
	return ""
}

// nssInstall ajoute la CA dans une base NSS si certutil est présent.
func nssInstall(bin, db, certPath, nick string) error {
	return exec.Command(bin, "-A", "-n", nick, "-t", "C,,", "-i", certPath, "-d", "sql:"+db).Run()
}

// nssHas : la base NSS contient cette CA (même empreinte).
func nssHas(bin, db, nick, wantFP string) bool {
	out, err := exec.Command(bin, "-L", "-n", nick, "-a", "-d", "sql:"+db).Output()
	if err != nil {
		return false
	}
	for rest := out; ; {
		var blk *pem.Block
		blk, rest = pem.Decode(rest)
		if blk == nil {
			return false
		}
		if fpSHA256(blk.Bytes) == wantFP {
			return true
		}
	}
}

// Setup : installe la CA et configure le proxy. Peut afficher des fenêtres
// d'autorisation du système (mot de passe macOS, avertissement Windows).
func Setup(dataDir, certPath, nick, proxyAddr string) Report {
	r := InstallCA(certPath, nick)
	if len(r.Errors) > 0 && len(r.CA) == 0 {
		return r
	}
	p := EnableProxy(dataDir, proxyAddr)
	r.CA = append(r.CA, p.CA...)
	r.Proxy, r.Manual, r.Errors = p.Proxy, append(r.Manual, p.Manual...), append(r.Errors, p.Errors...)
	return r
}

// InstallCA : étape 1 de l'assistant. CA locale (limitée aux .t2c) de confiance
// pour le système et les navigateurs Gecko.
func InstallCA(certPath, nick string) Report {
	var r Report
	der, err := certDER(certPath)
	if err != nil {
		r.errf("CA: %v", err)
		return r
	}
	installCA(&r, certPath, der, nick)
	if g := geckoPrefs(true); len(g) > 0 {
		r.CA = append(r.CA, "Firefox/Zen lisent les CA du système ("+strings.Join(g, ", ")+")")
	}
	for _, p := range GeckoProfiles() {
		if strings.Contains(strings.ToLower(p), "librewolf") && certutilBin() == "" {
			r.manual("%s : importe toi-même le certificat (Paramètres > Certificats) ; T2C ne modifie pas LibreWolf", profileLabel(p))
		}
	}
	return r
}

// EnableProxy : étape 3 de l'assistant. PAC (seuls les *.t2c passent par T2C).
// Refait le réglage Gecko : un Firefox lancé pour la première fois à l'étape 2
// a maintenant un profil.
func EnableProxy(dataDir, proxyAddr string) Report {
	var r Report
	if g := geckoPrefs(true); len(g) > 0 {
		r.CA = append(r.CA, "Firefox/Zen lisent les CA du système ("+strings.Join(g, ", ")+")")
	}
	configureProxy(&r, dataDir, PACURL(proxyAddr))
	_ = os.WriteFile(markerPath(dataDir), []byte("1\n"), 0o600)
	return r
}

// Restore remet les réglages proxy d'origine (la CA reste installée : sans
// danger, elle ne peut signer que des .t2c).
func Restore(dataDir string) error {
	if _, err := os.Stat(statePath(dataDir)); err != nil {
		return nil // rien à restaurer
	}
	return restoreProxy(dataDir)
}

// CleanStale : au démarrage, un PAC laissé par un arrêt brutal (réglages
// sauvegardés encore présents) alors que le proxy local ne répond pas est
// retiré : sinon le navigateur, ne trouvant pas le PAC, passerait les .t2c
// en connexion directe (fuite DNS) ou vers un autre programme du port.
// Rien n'est touché si une autre instance de l'app sert le proxy.
func CleanStale(dataDir, proxyAddr string) bool {
	if _, err := os.Stat(statePath(dataDir)); err != nil {
		return false
	}
	if c, err := net.DialTimeout("tcp", proxyAddr, 500*time.Millisecond); err == nil {
		c.Close()
		return false
	}
	return restoreFn(dataDir) == nil
}

// restoreFn : Restore (remplacée dans les tests, qui ne touchent pas au système).
var restoreFn = Restore

// Status : où la CA est de confiance, et si le PAC est actif.
func Status(certPath, nick, proxyAddr string) Report {
	var r Report
	der, err := certDER(certPath)
	if err != nil {
		return r
	}
	r.CA = caLocations(der, nick)
	r.Proxy = proxyLocations(PACURL(proxyAddr))
	return r
}

func markerPath(dataDir string) string { return filepath.Join(dataDir, "sysint-enabled") }

// Enabled : la configuration automatique a déjà été acceptée sur ce poste.
func Enabled(dataDir string) bool {
	_, err := os.Stat(markerPath(dataDir))
	return err == nil
}

// Disable : désactive la configuration automatique et restaure les réglages.
func Disable(dataDir string) error {
	_ = os.Remove(markerPath(dataDir))
	geckoPrefs(false)
	return Restore(dataDir)
}

// Reapply : au démarrage, remet le PAC si l'utilisateur l'avait activé
// (aucune nouvelle demande pour la CA, déjà installée).
func Reapply(dataDir, proxyAddr string) Report {
	var r Report
	configureProxy(&r, dataDir, PACURL(proxyAddr))
	return r
}
