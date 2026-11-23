//go:build windows

// Sonde de l'essai MSIX (workflow msix-essai.yml) : fait les mêmes écritures
// que pkg/sysint sous Windows (AutoConfigURL dans HKCU, certutil -user), puis
// note ce qu'elle a fait dans C:\msix-essai\resultat-<marque>.txt. Le
// workflow vérifie ensuite, hors du paquet, si ces écritures sont visibles.
//
// Le certificat va dans le magasin « My » de l'utilisateur et non « Root » :
// même stockage (HKCU\Software\Microsoft\SystemCertificates), sans la
// fenêtre de confirmation qui bloquerait une machine sans écran.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

const inetKey = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

func main() {
	marque := "paquet"
	if len(os.Args) > 1 {
		marque = os.Args[1]
	}
	var out strings.Builder
	fmt.Fprintf(&out, "marque=%s\n", marque)

	pac := "http://127.0.0.1:1/essai-" + marque + ".pac"
	k, _, err := registry.CreateKey(registry.CURRENT_USER, inetKey, registry.SET_VALUE)
	if err == nil {
		err = k.SetStringValue("AutoConfigURL", pac)
		k.Close()
	}
	fmt.Fprintf(&out, "pac=%s\npac_erreur=%v\n", pac, err)

	der, err := certificat(marque)
	if err == nil {
		f := filepath.Join(os.TempDir(), "essai-"+marque+".cer")
		if err = os.WriteFile(f, der, 0o600); err == nil {
			var b []byte
			b, err = exec.Command("certutil.exe", "-user", "-f", "-addstore", "My", f).CombinedOutput()
			fmt.Fprintf(&out, "certutil=%q\n", strings.TrimSpace(string(b)))
		}
		sum := sha1.Sum(der)
		fmt.Fprintf(&out, "empreinte=%s\n", strings.ToUpper(hex.EncodeToString(sum[:])))
	}
	fmt.Fprintf(&out, "cert_erreur=%v\n", err)

	_ = os.MkdirAll(`C:\msix-essai`, 0o755)
	_ = os.WriteFile(`C:\msix-essai\resultat-`+marque+".txt", []byte(out.String()), 0o644)
}

func certificat(marque string) ([]byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "T2C essai MSIX " + marque},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	return x509.CreateCertificate(rand.Reader, tpl, tpl, &key.PublicKey, key)
}
