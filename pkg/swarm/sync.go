package swarm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	mh "github.com/multiformats/go-multihash"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"sync"
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

			// Announce to DHT
			sm.AnnounceSite(ctx, pubKeyHash)
			fmt.Printf("📡 Swarm: Reprise du seeding pour le site %s\n", pubKeyHash)
		}
	}()

	return sm
}

func (sm *SwarmManager) handleStream(stream network.Stream) {
	defer stream.Close()
	if sm.DisableSeeding {
		return // Seeding is disabled, do not respond to requests
	}
	reader := bufio.NewReader(stream)
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

// AnnounceSite tells the DHT that we host this site.
func (sm *SwarmManager) AnnounceSite(ctx context.Context, pubKeyHash string) error {
	if sm.DisableSeeding {
		return nil // Seeding is disabled, do not announce
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
