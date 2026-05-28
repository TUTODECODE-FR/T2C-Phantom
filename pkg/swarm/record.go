package swarm

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
)

type RoutingRecord struct {
	Timestamp int64  `json:"timestamp"`
	ZipHash   string `json:"zip_hash"`
	Signature string `json:"signature"`
}

func (r *RoutingRecord) Sign(priv crypto.PrivKey) error {
	data := fmt.Sprintf("%d:%s", r.Timestamp, r.ZipHash)
	sig, err := priv.Sign([]byte(data))
	if err != nil {
		return err
	}
	r.Signature = base64.StdEncoding.EncodeToString(sig)
	return nil
}

func (r *RoutingRecord) Verify(pub crypto.PubKey) (bool, error) {
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

func CreateRecord(priv crypto.PrivKey, zipBytes []byte) (*RoutingRecord, error) {
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
