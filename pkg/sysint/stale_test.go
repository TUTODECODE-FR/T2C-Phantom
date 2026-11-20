package sysint

import (
	"net"
	"os"
	"testing"
)

// PAC resté actif après un arrêt brutal : retiré si le proxy ne répond pas,
// gardé si une autre instance le sert.
func TestCleanStale(t *testing.T) {
	dir := t.TempDir()
	restored := 0
	restoreFn = func(string) error { restored++; return nil }
	t.Cleanup(func() { restoreFn = Restore })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	if CleanStale(dir, addr) || restored != 0 {
		t.Fatal("rien à retirer sans état sauvegardé")
	}
	_ = os.WriteFile(statePath(dir), []byte(`{"pac":""}`), 0o600)
	if CleanStale(dir, addr) || restored != 0 {
		t.Fatal("PAC retiré alors que le proxy répond")
	}
	ln.Close()
	if !CleanStale(dir, addr) || restored != 1 {
		t.Fatal("PAC orphelin non retiré")
	}
}

// enterprise_roots : jamais avec certutil (CA ajoutée à la base NSS), jamais dans LibreWolf.
func TestGeckoNeedsPref(t *testing.T) {
	ff := "/home/x/.mozilla/firefox/abcd.default"
	lw := "/home/x/Library/Application Support/LibreWolf/Profiles/abcd.default"
	if !geckoNeedsPref(ff, false) || geckoNeedsPref(ff, true) || geckoNeedsPref(lw, false) || geckoNeedsPref(lw, true) {
		t.Fatal("règle enterprise_roots inexacte")
	}
}
