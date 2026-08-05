package p2p

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"t2c/pkg/crypto"
	"t2c/pkg/metrics"
	"t2c/pkg/utils"
)

// ReputationManager tracks Peer behaviors to prevent Leeching and Churn attacks.
type ReputationManager struct {
	mu           sync.RWMutex
	uploaded     map[string]int64
	downloaded   map[string]int64
	disconnects  map[string]int
	backoffUntil map[string]time.Time
	banned       map[string]bool // Liste des pairs bannis définitivement (Consensus / Triche)
}

func NewReputationManager() *ReputationManager {
	return &ReputationManager{
		uploaded:     make(map[string]int64),
		downloaded:   make(map[string]int64),
		disconnects:  make(map[string]int),
		backoffUntil: make(map[string]time.Time),
		banned:       make(map[string]bool),
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

// IsChoked returns true if the peer is leeching (ratio < 0.1), in backoff, or banned.
func (r *ReputationManager) IsChoked(peerID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Permanent Ban Check
	if r.banned[peerID] {
		return true
	}

	// 2. Churn Penalty (Exponential Backoff)
	if until, exists := r.backoffUntil[peerID]; exists {
		if time.Now().Before(until) {
			return true // Ignore the peer
		}
	}

	// 3. Tit-for-Tat (Anti-Leeching)
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

// BanPeer bannit définitivement un pair et incrémente les métriques SOC.
func (r *ReputationManager) BanPeer(peerID string, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if !r.banned[peerID] {
		r.banned[peerID] = true
		metrics.BannedPeers.Inc()
		utils.LogSecurityEvent("PEER_BANNED", fmt.Sprintf("Peer banned. Reason: %s", reason), peerID)
	}
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
	Manifest       *crypto.Manifest
	HTTPFallback   string // URL du LXC Proxy
	Reputation     *ReputationManager
	DisconnectPeer func(peerID string) // Callback pour couper la connexion libp2p physique
}

func NewDownloader(manifest *crypto.Manifest, fallbackURL string, disconnectCallback func(peerID string)) (*Downloader, error) {
	// 🛡️ SUPPLY CHAIN DEFENSE: Le Manifest DOIT être signé par la clé maître hors-ligne de l'association.
	if !crypto.VerifyManifestSignature(manifest) {
		return nil, errors.New("FATAL: Manifest signature is invalid. Possible supply chain attack")
	}

	return &Downloader{
		Manifest:       manifest,
		HTTPFallback:   fallbackURL,
		Reputation:     NewReputationManager(),
		DisconnectPeer: disconnectCallback,
	}, nil
}

// DownloadChunk tente de télécharger un fragment via P2P. 
// En cas d'échec ou de hash invalide, il fait un failover automatique.
func (d *Downloader) DownloadChunk(chunkIndex int, peerID string, data []byte) error {
	if d.Reputation.IsChoked(peerID) {
		return errors.New("peer is choked due to leeching, churn or ban penalty")
	}

	d.Reputation.TrackDownload(peerID, int64(len(data)))

	expectedHash := d.Manifest.ChunkHashes[chunkIndex]

	// Vérification Zero-Trust en RAM (Anti-TOCTOU & Merkle)
	if !crypto.ValidateChunk(data, expectedHash) {
		log.Printf("[ZERO-TRUST] Hash mismatch for chunk %d from peer %s", chunkIndex, peerID)
		
		// 📊 [SOC] Incrémenter la métrique de blocs corrompus et logguer l'anomalie
		metrics.CorruptedBlocks.Inc()
		utils.LogSecurityEvent("CORRUPTED_BLOCK", fmt.Sprintf("Merkle tree chunk validation failed for chunk %d", chunkIndex), peerID)

		// 1. Bannissement instantané et définitif du pair malveillant
		d.Reputation.BanPeer(peerID, fmt.Sprintf("Sent corrupted chunk %d", chunkIndex))

		// 2. Déconnexion immédiate au niveau réseau libp2p
		if d.DisconnectPeer != nil {
			d.DisconnectPeer(peerID)
		}

		// 3. Envoi du rapport de triche au tracker central
		d.ReportPeer(peerID)

		// 4. Failover HTTP transparent (Rien n'est écrit sur le disque)
		return d.DownloadChunkFailover(chunkIndex)
	}

	log.Printf("[ZERO-TRUST] Chunk %d validated successfully.", chunkIndex)
	return nil
}

func (d *Downloader) ReportPeer(peerID string) {
	log.Printf("[IMMUNE SYSTEM] Reporting malicious peer: %s", peerID)
	
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
	
	// 📊 [SOC] Incrémenter la métrique de failovers P2P -> HTTP
	metrics.Failovers.Inc()
	
	// En production, cette méthode télécharge le fragment en clair depuis le serveur de repli sécurisé.
	return errors.New("not implemented yet")
}
