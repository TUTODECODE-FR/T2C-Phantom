// t2c : client T2C Phantom (daemon + proxy .t2c + tableau de bord).
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"t2c/pkg/api"
	"t2c/pkg/metrics"
	"t2c/pkg/node"
	"t2c/pkg/peerid"
	"t2c/pkg/proxy"
	"t2c/pkg/proxy/assohttps"
	"t2c/pkg/sysint"
	"t2c/pkg/trust"
	"t2c/pkg/utils"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}
	_ = os.MkdirAll(utils.GetT2CDataDir(), 0o700)
	utils.InitSecurityLogger(utils.GetPath("security.log"))

	switch os.Args[1] {
	case "start":
		runStart(os.Args[2:])
	case "dashboard-client", "dashboard":
		if err := api.NewDashboard().Start(9090); err != nil {
			fmt.Println("tableau de bord:", err)
			os.Exit(1)
		}
	case "id":
		runID(os.Args[2:])
	case "doctor":
		runDoctor()
	case "setup", "unsetup":
		runSetup(os.Args[1] == "setup", os.Args[2:])
	default:
		printHelp()
		os.Exit(1)
	}
}

func dataDir() string { return utils.GetPath("peer") }

func runStart(args []string) {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	relay := fs.String("relay", envOr("T2C_RELAY_URL", node.DefaultRelay), "URL WebSocket du relais (Cloudflare → VPS)")
	proxyPort := fs.Int("proxy", 8080, "port du proxy local .t2c (0 = désactivé)")
	host := fs.String("host", "", "sites à héberger en tant que bénévole (séparés par des virgules) ; sans cette option, aucun site n'est hébergé")
	noVolunteer := fs.Bool("no-volunteer", false, "n'héberger aucune copie, même avec -host")
	carnetPath := fs.String("carnet", utils.GetPath("carnet.json"), "alias locaux (JSON alias.t2c → site)")
	metricsPort := fs.Int("metrics-port", 0, "métriques Prometheus sur 127.0.0.1:<port> (0 = off)")
	mintBits := fs.Int("mint-bits", peerid.DefaultMintBits, "PoW d'identité à la création")
	_ = fs.Parse(args)

	var hostSites []string
	if *host != "" {
		hostSites = strings.Split(*host, ",")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if _, err := os.Stat(filepath.Join(dataDir(), "identity.json")); os.IsNotExist(err) {
		fmt.Printf("Création de ton identité (clé aléatoire + PoW %d bits, quelques secondes)…\n", *mintBits)
	}
	n, err := node.Start(ctx, node.Config{
		RelayURL:  *relay,
		DataDir:   dataDir(),
		MintBits:  *mintBits,
		Volunteer: !*noVolunteer,
		HostSites: hostSites,
		Carnet:    api.LoadCarnet(*carnetPath),
	})
	if err != nil {
		fmt.Println("démarrage:", err)
		os.Exit(1)
	}
	fmt.Printf("Identité : %s (clé aléatoire ; ton IP n'est vue que du point d'entrée : Cloudflare, ou le relais en mode direct)\n", n.ID())
	fmt.Printf("Relais   : %s\n", *relay)
	utils.LogSecurityEvent("DAEMON_START", "client démarré", n.ID())

	if *proxyPort > 0 {
		go func() {
			if err := proxy.StartProxyServer(n, fmt.Sprintf("127.0.0.1:%d", *proxyPort)); err != nil {
				fmt.Println("proxy:", err)
			}
		}()
	}
	if *metricsPort > 0 {
		go metrics.StartMetricsServer(fmt.Sprintf("127.0.0.1:%d", *metricsPort))
	}

	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	var last node.Status
	for {
		select {
		case <-ctx.Done():
			n.Stop()
			return
		case <-t.C:
			st := n.Status()
			metrics.Observe(last, st)
			if st.Connected != last.Connected || st.LastError != last.LastError {
				state := "connecté"
				if !st.Connected {
					state = "déconnecté: " + st.LastError
				}
				fmt.Printf("[%s] %s · bénévole pour %d site(s) · morceaux pairs=%d origine=%d rejetés=%d\n",
					time.Now().Format("15:04:05"), state, len(st.Hosted), st.FromPeers, st.FromOrigin, st.BadChunks)
			}
			last = st
		}
	}
}

func runID(args []string) {
	fs := flag.NewFlagSet("id", flag.ExitOnError)
	mintBits := fs.Int("mint-bits", peerid.DefaultMintBits, "PoW d'identité à la création")
	_ = fs.Parse(args)
	id, err := peerid.LoadOrCreate(filepath.Join(dataDir(), "identity.json"), *mintBits)
	if err != nil {
		fmt.Println("identité:", err)
		os.Exit(1)
	}
	fmt.Println("Identité :", id.ID())
	fmt.Println("Fichier  :", filepath.Join(dataDir(), "identity.json"), "(0600 — ne le partage pas)")
	fmt.Println("Supprimer ce fichier = nouvelle identité (nouvelle PoW, score repart de zéro).")
}

// shortErr : message d'erreur réseau lisible (sans la pile d'adresses).
func shortErr(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "no such host"):
		return "nom inconnu (domaine pas encore actif ?)"
	case strings.Contains(msg, "timeout"):
		return "délai dépassé"
	case strings.Contains(msg, "refused"):
		return "connexion refusée"
	}
	return msg
}

func runDoctor() {
	fmt.Println("== Diagnostic T2C Phantom ==")
	if _, err := trust.RootKey(); err != nil {
		fmt.Println("[erreur] clé racine :", err)
	} else {
		fmt.Println("[ok] clé racine intégrée")
	}
	// relais : on teste une vraie connexion à chacun (un relais injoignable
	// n'est pas une panne tant qu'un autre répond)
	relays := node.DefaultRelays
	if u := os.Getenv("T2C_RELAY_URL"); u != "" {
		relays = append([]string{u}, relays...)
	}
	ok := 0
	for _, r := range relays {
		u, err := url.Parse(r)
		if err != nil {
			continue
		}
		host := u.Host
		if u.Port() == "" {
			host += ":443"
		}
		c, err := net.DialTimeout("tcp", host, 6*time.Second)
		if err != nil {
			fmt.Println("[info] relais", u.Hostname(), "injoignable :", shortErr(err))
			continue
		}
		c.Close()
		ok++
		fmt.Println("[ok] relais joignable :", u.Hostname())
	}
	if ok == 0 {
		fmt.Println("[erreur] aucun relais joignable : vérifie ta connexion Internet")
	}
	port := envOr("T2C_PROXY_PORT", "8080")
	if l, err := net.Listen("tcp", "127.0.0.1:"+port); err != nil {
		fmt.Printf("[attention] port %s déjà utilisé (T2C Phantom déjà lancé ? sinon T2C_PROXY_PORT=autre port)\n", port)
	} else {
		l.Close()
		fmt.Printf("[ok] port %s libre pour le proxy\n", port)
	}
	if err := os.MkdirAll(dataDir(), 0o700); err != nil {
		fmt.Println("[erreur] dossier de données :", err)
	} else {
		fmt.Println("[ok] dossier de données :", dataDir())
	}
}

// runSetup : configuration automatique des navigateurs (CA .t2c + proxy PAC), ou retour arrière.
func runSetup(enable bool, args []string) {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	port := fs.Int("proxy", 8080, "port du proxy local .t2c")
	_ = fs.Parse(args)
	dir := utils.GetT2CDataDir()
	if !enable {
		if err := sysint.Disable(dir); err != nil {
			fmt.Println("restauration:", err)
			os.Exit(1)
		}
		fmt.Println("Réglages proxy d'origine restaurés.")
		return
	}
	ca, err := assohttps.LoadOrCreate()
	if err != nil {
		fmt.Println("CA:", err)
		os.Exit(1)
	}
	rep := sysint.Setup(dir, ca.CertPEMPath(), assohttps.Nick(), fmt.Sprintf("127.0.0.1:%d", *port))
	for _, l := range rep.CA {
		fmt.Println("[ok] CA :", l)
	}
	for _, l := range rep.Proxy {
		fmt.Println("[ok] proxy .t2c :", l)
	}
	for _, l := range rep.Manual {
		fmt.Println("[à faire]", l)
	}
	for _, l := range rep.Errors {
		fmt.Println("[erreur]", l)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func printHelp() {
	fmt.Println(`T2C Phantom — client

Usage :
  t2c start [options]     Lance le client (connexion au relais + proxy .t2c)
  t2c dashboard           Tableau de bord web (http://127.0.0.1:9090)
  t2c id                  Affiche (ou crée) ton identité pseudonyme
  t2c doctor              Diagnostic
  t2c setup               Configure tous les navigateurs (CA .t2c + proxy automatique)
  t2c unsetup             Remet les réglages proxy d'origine

Options de start :
  -relay <url>            Relais (défaut wss://t2cphantom.eu.org/v1/ws)
  -proxy <port>           Proxy local .t2c (défaut 8080, 0 = off)
  -host a,b               Héberger ces sites dès le démarrage
  -no-volunteer           Ne pas héberger les sites visités
  -carnet <fichier>       Alias locaux
  -metrics-port <port>    Métriques Prometheus sur 127.0.0.1`)
}
