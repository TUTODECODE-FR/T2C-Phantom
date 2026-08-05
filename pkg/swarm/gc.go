package swarm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"t2c/pkg/utils"
	"time"
)

type SiteUsage struct {
	PubKeyHash string
	Path       string
	ZipPath    string
	Size       int64
	LastAccess time.Time
}

func (sm *SwarmManager) StartGarbageCollector(maxBytes, targetBytes int64, checkInterval time.Duration) {
	go func() {
		for {
			time.Sleep(checkInterval)
			sm.RunGC(maxBytes, targetBytes)
		}
	}()
}

func (sm *SwarmManager) RunGC(maxBytes, targetBytes int64) {
	sitesDir := filepath.Join(utils.GetT2CDataDir(), "sites")
	var totalSize int64
	var sites []SiteUsage

	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pubKeyHash := entry.Name()
		sitePath := filepath.Join(sitesDir, pubKeyHash)

		// Find the actual ZipHash from the local record.json
		var zipHash string
		recordPath := filepath.Join(sitePath, "record.json")
		if recordData, err := os.ReadFile(recordPath); err == nil {
			var rec RoutingRecord
			if json.Unmarshal(recordData, &rec) == nil {
				zipHash = rec.ZipHash
			}
		}

		// Fallback to memory records cache if record.json is missing
		if zipHash == "" {
			sm.Mutex.Lock()
			if rec, ok := sm.Records[pubKeyHash]; ok {
				zipHash = rec.ZipHash
			}
			sm.Mutex.Unlock()
		}

		zipPath := ""
		var zipSize int64
		if zipHash != "" {
			zipPath = filepath.Join(sm.ZipStorage, zipHash+".zip")
			if info, err := os.Stat(zipPath); err == nil {
				zipSize = info.Size()
			}
		}

		// Calculate size of site dir
		var siteSize int64
		filepath.Walk(sitePath, func(_ string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				siteSize += info.Size()
			}
			return nil
		})

		// Get Last Access time (using dir's mod time as marker)
		info, err := os.Stat(sitePath)
		if err != nil {
			continue
		}

		totalSize += (siteSize + zipSize)
		sites = append(sites, SiteUsage{
			PubKeyHash: pubKeyHash,
			Path:       sitePath,
			ZipPath:    zipPath,
			Size:       siteSize + zipSize,
			LastAccess: info.ModTime(),
		})
	}

	if totalSize <= maxBytes {
		return // All good, under quota
	}

	fmt.Printf("🧹 Swarm GC Triggered: Total size %d MB exceeds quota %d MB. Cleaning up...\n", totalSize/1024/1024, maxBytes/1024/1024)

	// Sort by oldest first (Least Recently Used)
	sort.Slice(sites, func(i, j int) bool {
		return sites[i].LastAccess.Before(sites[j].LastAccess)
	})

	for _, site := range sites {
		// Stop seeding
		sm.Mutex.Lock()
		delete(sm.Records, site.PubKeyHash)
		sm.Mutex.Unlock()

		// Delete files
		os.RemoveAll(site.Path)
		os.Remove(site.ZipPath)

		totalSize -= site.Size
		fmt.Printf("🗑️  Swarm GC: Evicted site %s (Freed %d MB)\n", site.PubKeyHash, site.Size/1024/1024)

		if totalSize <= targetBytes {
			fmt.Printf("✅ Swarm GC: Target reached. Current size: %d MB\n", totalSize/1024/1024)
			break
		}
	}
}
