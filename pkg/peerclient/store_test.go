package peerclient

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"t2c/pkg/trust"
)

func TestContentStoreEncryptedAtRest(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	data := bytes.Repeat([]byte("TUTODECODE-secret-lisible "), 30000) // ~780 Ko → plusieurs morceaux
	m, err := trust.BuildManifest("demo", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	_ = m.Sign(priv)
	dir := t.TempDir()
	s, _ := NewContentStore(dir)
	if err := s.Put(m, data); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, m.Root+".enc"))
	if bytes.Contains(raw, []byte("TUTODECODE-secret")) {
		t.Fatal("contenu en clair sur le disque")
	}
	// redémarrage : relecture et déchiffrement
	s2, _ := NewContentStore(dir)
	if _, err := s2.Load(m.Root, pub); err != nil {
		t.Fatal(err)
	}
	got, err := s2.Read(m.Root)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("relecture: %v", err)
	}
	// altération sur le disque : le morceau n'est plus servi
	raw[10] ^= 1
	_ = os.WriteFile(filepath.Join(dir, m.Root+".enc"), raw, 0o600)
	if _, err := s2.Chunk(m.Root, 0); err == nil {
		t.Fatal("morceau altéré servi")
	}
	if _, err := s2.Chunk(m.Root, len(m.Chunks)-1); err != nil {
		t.Fatalf("dernier morceau intact refusé: %v", err)
	}
	// ancienne copie en clair : chiffrée au chargement puis supprimée
	legacy := t.TempDir()
	mb, _ := os.ReadFile(filepath.Join(dir, m.Root+".json"))
	_ = os.WriteFile(filepath.Join(legacy, m.Root+".json"), mb, 0o600)
	_ = os.WriteFile(filepath.Join(legacy, m.Root+".blob"), data, 0o600)
	s3, _ := NewContentStore(legacy)
	if _, err := s3.Load(m.Root, pub); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(legacy, m.Root+".blob")); err == nil {
		t.Fatal("copie en clair conservée")
	}
	if got, _ := s3.Read(m.Root); !bytes.Equal(got, data) {
		t.Fatal("migration incorrecte")
	}
	s3.Remove(m.Root)
	if len(s3.Roots()) != 0 || s3.Size() != 0 {
		t.Fatal("suppression incomplète")
	}
}
