package p2p

import (
	"context"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"t2c/pkg/crypto"
	"t2c/pkg/metrics"
)

// ReputationManager tracks Peer behaviors to prevent Leeching and Churn attacks.
type ReputationManager struct {
	mu           sync.RWMutex
	uploaded     map[string]int64
	downloaded   map[string]int64
	disconnects  map[string]int
	backoffUntil map[string]time.Time
}

func NewReputationManager() *ReputationManager {
	return &ReputationManager{
		uploaded:     make(map[string]int64),
		downloaded:   make(map[string]int64),
		disconnects:  make(map[string]int),
		backoffUntil: make(map[string]time.Time),
	}
}

// TrackUpload records bytes sent to a peer.
func (r *ReputationManager) TrackUpload(peerID string, bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.uploaded[peerID] += bytes
}

// TrackDownload records bytes received from a peer.
func (r *ReputationManager) TrackDownload(peerID string, bytes int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.downloaded[peerID] += bytes
}

// IsChoked returns true if the peer is leeching (ratio < 0.1) or in backoff.
func (r *ReputationManager) IsChoked(peerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Churn Penalty (Exponential Backoff)
	if until, exists := r.backoffUntil[peerID]; exists {
		if time.Now().Before(until) {
			return true // Ignore the peer
		}
	}

	// 2. Tit-for-Tat (Anti-Leeching)
	up := r.uploaded[peerID]
	down := r.downloaded[peerID]
	
	// Grace threshold: Give them a chance before enforcing ratio (e.g. 5 MB)
	if down > 5*1024*1024 {
		ratio := float64(up) / float64(down)
		if ratio < 0.1 {
			return true // Choked
		}
	}
	return false
}

// HandleDisconnect penalizes peers that drop connection unexpectedly (Churn attack).
func (r *ReputationManager) HandleDisconnect(peerID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disconnects[peerID]++
	
	// Exponential backoff: 2^n seconds
	penalty := time.Duration(1<<r.disconnects[peerID]) * time.Second
	if penalty > 1*time.Hour {
		penalty = 1 * time.Hour
	}
	r.backoffUntil[peerID] = time.Now().Add(penalty)
	log.Printf("[ANTI-CHURN] Peer %s disconnected unexpectedly. Ignored for %v", peerID, penalty)
}

// Downloader gère la récupération des blocs P2P et la fallback
type Downloader struct {
	Manifest     *crypto.Manifest
	HTTPFallback string // URL du LXC Proxy
	Reputation   *ReputationManager
}

func NewDownloader(manifest *crypto.Manifest, fallbackURL string) (*Downloader, error) {
	// 🛡️ SUPPLY CHAIN DEFENSE: Le Manifest DOIT être signé par la clé maître hors-ligne.
	if !crypto.VerifyManifestSignature(manifest) {
		return nil, errors.New("FATAL: Manifest signature is invalid. Possible supply chain attack")
	}

	return &Downloader{
		Manifest:     manifest,
		HTTPFallback: fallbackURL,
		Reputation:   NewReputationManager(),
	}, nil
}

// DownloadChunk tente de télécharger un fragment via P2P. 
// En cas d'échec ou de hash invalide, il peut faire un failover.
func (d *Downloader) DownloadChunk(chunkIndex int, peerID string, data []byte) error {
	if d.Reputation.IsChoked(peerID) {
		return errors.New("peer is choked due to leeching or churn penalty")
	}

	d.Reputation.TrackDownload(peerID, int64(len(data)))

	expectedHash := d.Manifest.ChunkHashes[chunkIndex]

	// Vérification Zero-Trust en RAM
	if !crypto.ValidateChunk(data, expectedHash) {
		log.Printf("[ZERO-TRUST] Hash mismatch for chunk %d from peer %s", chunkIndex, peerID)
		
		// 📊 [SOC] Incrémenter la métrique des attaques TOCTOU
		metrics.CorruptedBlocks.Inc()
		
		// 1. Déconnexion immédiate du peer (à intégrer avec libp2p Host)
		// 2. Envoi d'un rapport de ban au serveur
		d.ReportPeer(peerID)

		// 3. Failover HTTP transparent
		return d.DownloadChunkFailover(chunkIndex)
	}

	log.Printf("[ZERO-TRUST] Chunk %d validated successfully.", chunkIndex)
	return nil
}

func (d *Downloader) ReportPeer(peerID string) {
	log.Printf("[IMMUNE SYSTEM] Reporting malicious peer: %s", peerID)
	
	// Execute in a background goroutine to not block the downloader
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "POST", d.HTTPFallback+"/report", nil)
		if err != nil {
			return
		}
		
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[IMMUNE SYSTEM] Failed to report peer: %v", err)
			return
		}
		defer resp.Body.Close()
	}()
}

func (d *Downloader) DownloadChunkFailover(chunkIndex int) error {
	log.Printf("[FAILOVER] Downloading chunk %d directly from LXC Proxy (%s)", chunkIndex, d.HTTPFallback)
	
	// 📊 [SOC] Incrémenter la métrique des failovers P2P -> HTTP
	metrics.Failovers.Inc()
	
	// Code de téléchargement HTTP classique
	return errors.New("not implemented yet")
}
