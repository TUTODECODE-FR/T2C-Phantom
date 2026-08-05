package swarm

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	t2ccrypto "t2c/pkg/crypto"
)

type RoutingRecord struct {
	Timestamp int64  `json:"timestamp"`
	ZipHash   string `json:"zip_hash"`
	Signature string `json:"signature"`
}

func (r *RoutingRecord) Sign(priv libp2pcrypto.PrivKey) error {
	data := fmt.Sprintf("%d:%s", r.Timestamp, r.ZipHash)
	sig, err := priv.Sign([]byte(data))
	if err != nil {
		return err
	}
	r.Signature = base64.StdEncoding.EncodeToString(sig)
	return nil
}

func (r *RoutingRecord) Verify(pub libp2pcrypto.PubKey) (bool, error) {
	data := fmt.Sprintf("%d:%s", r.Timestamp, r.ZipHash)
	sigBytes, err := base64.StdEncoding.DecodeString(r.Signature)
	if err != nil {
		return false, err
	}
	return pub.Verify([]byte(data), sigBytes)
}

func HashFileBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func HashReader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func HashLocalFile(path string) (string, error) {
	// Tente de déchiffrer en mémoire pour calculer le hash du fichier d'origine
	decrypted, err := t2ccrypto.ReadDecryptedFile(path)
	if err == nil {
		return HashFileBytes(decrypted), nil
	}
	
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return HashReader(f)
}

func CreateRecord(priv libp2pcrypto.PrivKey, zipBytes []byte) (*RoutingRecord, error) {
	hash := HashFileBytes(zipBytes)
	record := &RoutingRecord{
		Timestamp: time.Now().Unix(),
		ZipHash:   hash,
	}
	err := record.Sign(priv)
	return record, err
}

func SerializeRecord(record *RoutingRecord) ([]byte, error) {
	return json.Marshal(record)
}

func DeserializeRecord(data []byte) (*RoutingRecord, error) {
	var r RoutingRecord
	err := json.Unmarshal(data, &r)
	return &r, err
}
