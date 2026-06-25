package swarm

import (
	"archive/zip"
	"bufio"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"t2c/pkg/utils"
	"time"

	"github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
)

const SyncProtocolID = "/t2c/swarm/1.0"

type SyncRequest struct {
	Action string `json:"action"` // "get_record" or "get_zip"
	Key    string `json:"key"`    // PubKeyHash or ZipHash
}

type SwarmManager struct {
	Host           host.Host
	DHT            *dht.IpfsDHT
	Records        map[string]*RoutingRecord // PubKeyHash -> Record
	ZipStorage     string                    // Directory to store Zips (e.g. data/swarm_zips)
	LastChecked    map[string]time.Time      // PubKeyHash -> Last verification time
	Syncing        map[string]bool           // PubKeyHash -> Is currently syncing
	DisableSeeding bool                      // If true, do not announce or upload files to others
	Mutex          sync.Mutex
}

func NewSwarmManager(h host.Host, d *dht.IpfsDHT, zipStorage string) *SwarmManager {
	os.MkdirAll(zipStorage, 0755)
	sm := &SwarmManager{
		Host:        h,
		DHT:         d,
		Records:     make(map[string]*RoutingRecord),
		ZipStorage:  zipStorage,
		LastChecked: make(map[string]time.Time),
		Syncing:     make(map[string]bool),
	}
	h.SetStreamHandler(SyncProtocolID, sm.handleStream)
	
	// Start GC: Max 1 GB (1024*1024*1024), Target 800 MB (800*1024*1024), Check every 10 mins
	sm.StartGarbageCollector(1024*1024*1024, 800*1024*1024, 10*time.Minute)

	// Load and seed existing sites in the background
	go func() {
		sitesDir := filepath.Join("data", "sites")
		entries, err := os.ReadDir(sitesDir)
		if err != nil {
			return
		}

		ctx := context.Background()
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			pubKeyHash := entry.Name()
			recordPath := filepath.Join(sitesDir, pubKeyHash, "record.json")
			
			recordData, err := os.ReadFile(recordPath)
			if err != nil {
				continue
			}

			record, err := DeserializeRecord(recordData)
			if err != nil {
				continue
			}

			sm.Mutex.Lock()
			sm.Records[pubKeyHash] = record
			sm.Mutex.Unlock()

			fmt.Printf("📡 Swarm: Reprise du seeding pour le site %s\n", pubKeyHash)
		}

		// Loop permanently to announce sites every 2 minutes
		for {
			time.Sleep(5 * time.Second) // wait for relay connection on startup
			sm.Mutex.Lock()
			var hashes []string
			for hash := range sm.Records {
				hashes = append(hashes, hash)
			}
			sm.Mutex.Unlock()

			for _, h := range hashes {
				sm.AnnounceSite(ctx, h)
			}
			time.Sleep(2 * time.Minute)
		}
	}()

	return sm
}

func (sm *SwarmManager) handleStream(stream network.Stream) {
	defer stream.Close()
	if sm.DisableSeeding {
		return // Seeding is disabled, do not respond to requests
	}
	
	// Anti-Slowloris / Goroutine Leak protection
	stream.SetReadDeadline(time.Now().Add(10 * time.Second))
	
	limitReader := io.LimitReader(stream, 4096)
	reader := bufio.NewReader(limitReader)
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	var req SyncRequest
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		return
	}

	switch req.Action {
	case "get_record":
		rec, ok := sm.Records[req.Key]
		if ok {
			data, _ := SerializeRecord(rec)
			stream.Write(data)
			stream.Write([]byte("\n"))
		}
	case "get_zip":
		zipPath := fmt.Sprintf("%s/%s.zip", sm.ZipStorage, req.Key)
		file, err := os.Open(zipPath)
		if err == nil {
			defer file.Close()
			io.Copy(stream, file)
		}
	}
}

func GetCIDForString(str string) (cid.Cid, error) {
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   mh.SHA2_256,
		MhLength: -1, // default length
	}
	return pref.Sum([]byte(str))
}

// VerifySiteIntegrity hashes the local ZIP and compares every extracted file against the ZIP contents.
func VerifySiteIntegrity(sm *SwarmManager, pubKeyHash string, record *RoutingRecord) error {
	localZipPath := filepath.Join(sm.ZipStorage, record.ZipHash+".zip")
	
	// 1. Verify the ZIP hash (Immutability check) via streaming
	zipHashStr, err := HashLocalFile(localZipPath)
	if err != nil {
		return fmt.Errorf("could not read local zip: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(zipHashStr), []byte(record.ZipHash)) == 0 {
		utils.LogSecurityEvent("local_zip_hash_mismatch", "Local ZIP hash does not match record", pubKeyHash)
		return fmt.Errorf("local ZIP hash mismatch")
	}

	// 2. Open ZIP in streaming mode
	zipReader, err := zip.OpenReader(localZipPath)
	if err != nil {
		return fmt.Errorf("could not open local zip: %w", err)
	}
	defer zipReader.Close()

	siteDir := filepath.Join("data", "sites", pubKeyHash)
	
	// Create a map to track which files are in the zip
	zipFiles := make(map[string]bool)

	// 3. Compare each file in the ZIP to the extracted file on disk
	for _, f := range zipReader.File {
		if f.FileInfo().IsDir() {
			continue
		}
		
		// Prevent path traversal vulnerabilities locally
		if strings.Contains(f.Name, "..") {
			continue
		}
		
		zipFiles[f.Name] = true

		// Read file from ZIP via streaming
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %s", f.Name)
		}
		zipFileHash, err := HashReader(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to hash file in zip: %s", f.Name)
		}

		// Read file from disk via streaming
		diskPath := filepath.Join(siteDir, f.Name)
		diskFileHash, err := HashLocalFile(diskPath)
		if err != nil {
			return fmt.Errorf("missing extracted file: %s", f.Name)
		}

		// Compare hashes securely against Timing Attacks
		if subtle.ConstantTimeCompare([]byte(zipFileHash), []byte(diskFileHash)) == 0 {
			utils.LogSecurityEvent("extracted_file_tampered", fmt.Sprintf("File %s hash mismatch", f.Name), pubKeyHash)
			return fmt.Errorf("extracted file tampered: %s", f.Name)
		}
	}

	// 4. Verify no extra injected files (Side-channel risk)
	err = filepath.Walk(siteDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == "record.json" { // We inject record.json ourselves
			return nil
		}
		
		relPath, err := filepath.Rel(siteDir, path)
		if err != nil {
			return err
		}
		
		// Normalize paths for cross-platform (windows uses \)
		relPath = filepath.ToSlash(relPath)

		if !zipFiles[relPath] {
			utils.LogSecurityEvent("injected_malicious_file", fmt.Sprintf("File %s not in ZIP", relPath), pubKeyHash)
			return fmt.Errorf("injected malicious file detected: %s", relPath)
		}
		return nil
	})

	return err
}

// AnnounceSite tells the DHT that we host this site.
func (sm *SwarmManager) AnnounceSite(ctx context.Context, pubKeyHash string) error {
	if sm.DisableSeeding {
		return nil // Seeding is disabled, do not announce
	}
	
	sm.Mutex.Lock()
	record, ok := sm.Records[pubKeyHash]
	sm.Mutex.Unlock()

	if ok {
		if err := VerifySiteIntegrity(sm, pubKeyHash, record); err != nil {
			// Auditable alert!
			fmt.Printf("\n🚨 ALERTE SÉCURITÉ : Le module %s a été modifié localement !\n", pubKeyHash)
			fmt.Printf("Détail : %v\n", err)
			fmt.Println("Action : Réinitialisation vers la version officielle et arrêt de l'hébergement.")
			
			// Delete corrupted files
			os.Remove(filepath.Join(sm.ZipStorage, record.ZipHash+".zip"))
			os.RemoveAll(filepath.Join("data", "sites", pubKeyHash))
			
			sm.Mutex.Lock()
			delete(sm.Records, pubKeyHash)
			sm.Mutex.Unlock()
			
			return err
		}
	}

	id, err := GetCIDForString(pubKeyHash)
	if err != nil {
		return err
	}
	return sm.DHT.Provide(ctx, id, true)
}

// FindSeeders searches the DHT for peers hosting this site.
func (sm *SwarmManager) FindSeeders(ctx context.Context, pubKeyHash string) ([]peer.AddrInfo, error) {
	id, err := GetCIDForString(pubKeyHash)
	if err != nil {
		return nil, err
	}
	peers, err := sm.DHT.FindProviders(ctx, id)
	if err != nil {
		return nil, err
	}
	return peers, nil
}
