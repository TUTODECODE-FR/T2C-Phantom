package swarm

import (
	"bufio"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"t2c/pkg/crypto"
)

func SyncSite(ctx context.Context, sm *SwarmManager, h host.Host, pubKeyHash string) (string, error) {
	siteDir := filepath.Join("data", "sites", pubKeyHash)
	os.MkdirAll(siteDir, 0755)

	// Update mtime for LRU Garbage Collector
	now := time.Now()
	os.Chtimes(siteDir, now, now)

	sm.Mutex.Lock()
	lastChecked := sm.LastChecked[pubKeyHash]
	isSyncing := sm.Syncing[pubKeyHash]
	hasLocal := false
	if _, err := os.Stat(filepath.Join(siteDir, "index.html")); err == nil {
		hasLocal = true
	}

	// Si le site est localement disponible et a été vérifié il y a moins de 15 minutes,
	// ou si une synchronisation est déjà en cours, on le sert immédiatement sans requêter le réseau.
	if hasLocal && (time.Since(lastChecked) < 15*time.Minute || isSyncing) {
		sm.Mutex.Unlock()
		return siteDir, nil
	}

	if isSyncing {
		// S'il est en cours de synchronisation mais qu'on ne l'a pas localement,
		// on déverrouille et on laisse performSync s'exécuter pour le télécharger
		sm.Mutex.Unlock()
	} else {
		sm.Syncing[pubKeyHash] = true
		sm.Mutex.Unlock()
	}

	if hasLocal {
		// Si on a le site localement, on lance la vérification en arrière-plan
		go func() {
			err := performSync(context.Background(), sm, h, pubKeyHash, siteDir)
			sm.Mutex.Lock()
			sm.Syncing[pubKeyHash] = false
			if err == nil {
				sm.LastChecked[pubKeyHash] = time.Now()
			}
			sm.Mutex.Unlock()
		}()
		return siteDir, nil
	}

	// Si on n'a pas le site en local, on fait une mise à jour synchrone (bloquante)
	err := performSync(ctx, sm, h, pubKeyHash, siteDir)
	
	sm.Mutex.Lock()
	sm.Syncing[pubKeyHash] = false
	if err == nil {
		sm.LastChecked[pubKeyHash] = time.Now()
	}
	sm.Mutex.Unlock()

	return siteDir, err
}

func performSync(ctx context.Context, sm *SwarmManager, h host.Host, pubKeyHash, siteDir string) error {
	// 2. Find Seeders
	peers, err := sm.FindSeeders(ctx, pubKeyHash)
	if err != nil || len(peers) == 0 {
		return fmt.Errorf("No seeders found for site %s", pubKeyHash)
	}

	// 2. Connect to a seeder and get the Record
	var record *RoutingRecord
	var seederID peer.ID

	for _, p := range peers {
		if p.ID == h.ID() {
			continue
		}
		
		h.Connect(ctx, p)
		stream, err := h.NewStream(ctx, p.ID, SyncProtocolID)
		if err != nil {
			continue
		}
		
		req := SyncRequest{Action: "get_record", Key: pubKeyHash}
		reqBytes, _ := json.Marshal(req)
		stream.Write(reqBytes)
		stream.Write([]byte("\n"))
		
		reader := bufio.NewReader(stream)
		line, err := reader.ReadString('\n')
		if err == nil {
			rec, err := DeserializeRecord([]byte(line))
			if err == nil {
				record = rec
				seederID = p.ID
				stream.Close()
				break
			}
		}
		stream.Close()
	}

	if record == nil {
		return fmt.Errorf("Could not fetch routing record")
	}

	// 1. Verify record signature using public key derived from site address (pubKeyHash)
	pubKey, err := crypto.ParseT2CPublicKey(pubKeyHash)
	if err != nil {
		return fmt.Errorf("invalid site address: %w", err)
	}

	valid, err := record.Verify(pubKey)
	if err != nil || !valid {
		return fmt.Errorf("Security Error: Invalid RoutingRecord signature for site %s", pubKeyHash)
	}

	// 2. Check if we already have a newer or same version in memory
	sm.Mutex.Lock()
	existingRecord, exists := sm.Records[pubKeyHash]
	sm.Mutex.Unlock()
	if exists && record.Timestamp <= existingRecord.Timestamp {
		// The local version is already up to date, nothing to do.
		return nil
	}

	localZipPath := filepath.Join(sm.ZipStorage, record.ZipHash+".zip")
	if _, err := os.Stat(localZipPath); err == nil {
		zipHashStr, err := HashLocalFile(localZipPath)
		if err == nil {
			if subtle.ConstantTimeCompare([]byte(zipHashStr), []byte(record.ZipHash)) == 0 {
				// Corrupted or maliciously modified ZIP!
				os.Remove(localZipPath)
				os.RemoveAll(siteDir)
				fmt.Printf("🚨 Security Alert: Local ZIP hash mismatch for %s. Deleting and re-downloading.\n", pubKeyHash)
			} else {
				// We already have this VALID zip. Let's make sure it's extracted.
				if _, err := os.Stat(filepath.Join(siteDir, "index.html")); err != nil {
					// We need the bytes to pass to UnzipDirectory which still takes []byte
					if downloadedBytes, err := os.ReadFile(localZipPath); err == nil {
						UnzipDirectory(downloadedBytes, siteDir)
					}
				}
				// Save the record to the site directory
				if recordBytes, err := SerializeRecord(record); err == nil {
					os.WriteFile(filepath.Join(siteDir, "record.json"), recordBytes, 0644)
				}
				sm.Mutex.Lock()
				sm.Records[pubKeyHash] = record
				sm.Mutex.Unlock()
				sm.AnnounceSite(ctx, pubKeyHash)
				return nil
			}
		}
	}

	// 5. Download the ZIP
	stream, err := h.NewStream(ctx, seederID, SyncProtocolID)
	if err != nil {
		return err
	}
	defer stream.Close()

	req := SyncRequest{Action: "get_zip", Key: record.ZipHash}
	reqBytes, _ := json.Marshal(req)
	stream.Write(reqBytes)
	stream.Write([]byte("\n"))

	zipFile, err := os.Create(localZipPath)
	if err != nil {
		return err
	}
	io.Copy(zipFile, stream)
	zipFile.Close()

	// 5. Verify Hash
	downloadedBytes, _ := os.ReadFile(localZipPath)
	actualHash := HashFileBytes(downloadedBytes)
	if actualHash != record.ZipHash {
		os.Remove(localZipPath)
		return fmt.Errorf("Security Error: ZIP Hash mismatch! Expected %s, got %s", record.ZipHash, actualHash)
	}

	// 6. Extract
	UnzipDirectory(downloadedBytes, siteDir)
	
	// Save the record to the site directory
	if recordBytes, err := SerializeRecord(record); err == nil {
		os.WriteFile(filepath.Join(siteDir, "record.json"), recordBytes, 0644)
	}
	
	// 8. Seed it to the network!
	sm.Mutex.Lock()
	sm.Records[pubKeyHash] = record
	sm.Mutex.Unlock()
	sm.AnnounceSite(ctx, pubKeyHash)

	fmt.Printf("✅ Swarm Sync Complete: %s (Seeding now...)\n", pubKeyHash)
	return nil
}
