// t2c-peer : client minimal du réseau (en attendant l'intégration au daemon t2c).
//
//	t2c-peer -relay wss://t2cphantom.eu.org/v1/ws list
//	t2c-peer -relay ... fetch <site> <dossier-sortie>
//	t2c-peer -relay ... host  <site>        (bénévole : reste connecté et sert)
//
// L'identité (clé aléatoire + PoW) est stockée dans -data/identity.json.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"t2c/pkg/node"
	"t2c/pkg/peerclient"
	"t2c/pkg/peerid"
	"t2c/pkg/trust"
)

func die(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "t2c-peer: "+f+"\n", a...)
	os.Exit(1)
}

func main() {
	relay := flag.String("relay", "", "URL WebSocket du relais (défaut : relais par défaut du réseau, essayés dans l'ordre)")
	data := flag.String("data", defaultDataDir(), "dossier de données (identité, copies hébergées) ; défaut : $T2C_DATA_DIR/t2c-peer si la variable est définie")
	bits := flag.Int("mint-bits", peerid.DefaultMintBits, "PoW d'identité (à la création)")
	flag.Parse()
	relays = node.DefaultRelays
	if *relay != "" {
		relays = []string{*relay}
	}
	if flag.NArg() < 1 {
		die("usage: t2c-peer [options] <list|fetch|host> ...")
	}
	rootKey, err := trust.RootKey()
	if err != nil {
		die("%v", err)
	}

	idPath := filepath.Join(*data, "identity.json")
	if _, err := os.Stat(idPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "création de l'identité (PoW %d bits)…\n", *bits)
	}
	id, err := peerid.LoadOrCreate(idPath, *bits)
	if err != nil {
		die("%v", err)
	}
	fmt.Fprintf(os.Stderr, "identité : %s\n", id.ID())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	opt := peerclient.Options{RootKey: rootKey, Guard: trust.NewVersionGuard(filepath.Join(*data, "versions.json"))}

	switch flag.Arg(0) {
	case "list":
		c := dial(ctx, *relay, id, opt)
		defer c.Close()
		cat, err := c.Catalog(ctx)
		if err != nil {
			die("catalogue: %v", err)
		}
		fmt.Printf("catalogue v%d (signature vérifiée)\n", cat.Version)
		for _, s := range cat.Sites {
			hosts, _ := c.Find(ctx, s.Root)
			fmt.Printf("  %-20s %-30q %8d o  bénévoles en ligne: %d\n", s.ID, s.Title, s.Size, len(hosts))
		}

	case "fetch":
		if flag.NArg() != 3 {
			die("usage: t2c-peer fetch <site> <dossier>")
		}
		c := dial(ctx, *relay, id, opt)
		defer c.Close()
		cat, err := c.Catalog(ctx)
		if err != nil {
			die("catalogue: %v", err)
		}
		e, ok := cat.Find(flag.Arg(1))
		if !ok {
			die("site %q absent du catalogue", flag.Arg(1))
		}
		t0 := time.Now()
		blob, _, st, err := c.Fetch(ctx, e.Root)
		if err != nil {
			die("téléchargement: %v", err)
		}
		if err := node.Unzip(blob, flag.Arg(2)); err != nil {
			die("extraction: %v", err)
		}
		fmt.Printf("%s → %s en %s : %d morceaux de bénévoles, %d de l'origine, %d faux rejetés\n",
			e.ID, flag.Arg(2), time.Since(t0).Round(time.Millisecond),
			st.FromPeers.Load(), st.FromOrigin.Load(), st.BadChunks.Load())

	case "host":
		if flag.NArg() != 2 {
			die("usage: t2c-peer host <site>")
		}
		store, err := peerclient.NewContentStore(filepath.Join(*data, "hosted"))
		if err != nil {
			die("%v", err)
		}
		opt.Store = store
		var tickets []trust.Ticket
		for ctx.Err() == nil {
			c, err := dialAny(ctx, id, opt)
			if err != nil {
				fmt.Fprintf(os.Stderr, "connexion: %v — nouvel essai dans 10 s\n", err)
				sleep(ctx, 10*time.Second)
				continue
			}
			c.AdoptTickets(tickets)
			if t, err := c.Host(ctx, flag.Arg(1)); err != nil {
				fmt.Fprintf(os.Stderr, "bénévolat refusé: %v\n", err)
			} else {
				fmt.Printf("bénévole pour %s (racine %s…, ticket jusqu'au %s)\n", t.Site, t.Root[:12], time.Unix(t.Expires, 0).Format(time.DateTime))
			}
			select {
			case <-ctx.Done():
			case <-c.Done():
				fmt.Fprintln(os.Stderr, "déconnecté du relais, reconnexion…")
			}
			tickets = c.Tickets()
			_ = c.Close()
			sleep(ctx, 2*time.Second)
		}

	default:
		die("commande inconnue %q", flag.Arg(0))
	}
}

// relays : relais à essayer dans l'ordre (celui de -relay s'il est donné,
// sinon les relais par défaut, comme l'application).
var relays []string

func dial(ctx context.Context, _ string, id *peerid.Identity, opt peerclient.Options) *peerclient.Client {
	c, err := dialAny(ctx, id, opt)
	if err != nil {
		die("connexion au relais: %v", err)
	}
	return c
}

func dialAny(ctx context.Context, id *peerid.Identity, opt peerclient.Options) (*peerclient.Client, error) {
	var last error
	for _, u := range relays {
		dctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		c, err := peerclient.Dial(dctx, u, id, opt)
		cancel()
		if err == nil {
			return c, nil
		}
		last = fmt.Errorf("%s: %w", u, err)
	}
	return nil, last
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func defaultDataDir() string {
	if d := os.Getenv("T2C_DATA_DIR"); d != "" {
		return filepath.Join(d, "t2c-peer")
	}
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "t2c-peer")
	}
	return ".t2c-peer"
}
