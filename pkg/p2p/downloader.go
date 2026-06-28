package p2p

import (
	"errors"
	"log"
	"t2c/pkg/crypto"
	"t2c/pkg/metrics"
)

// Downloader gère la récupération des blocs P2P et la fallback
type Downloader struct {
	Manifest *crypto.Manifest
	HTTPFallback string // URL du LXC Proxy
}

func NewDownloader(manifest *crypto.Manifest, fallbackURL string) (*Downloader, error) {
	// 🛡️ SUPPLY CHAIN DEFENSE: Le Manifest DOIT être signé par la clé maître hors-ligne.
	if !crypto.VerifyManifestSignature(manifest) {
		return nil, errors.New("FATAL: Manifest signature is invalid. Possible supply chain attack")
	}

	return &Downloader{
		Manifest: manifest,
		HTTPFallback: fallbackURL,
	}, nil
}

// DownloadChunk tente de télécharger un fragment via P2P. 
// En cas d'échec ou de hash invalide, il peut faire un failover.
func (d *Downloader) DownloadChunk(chunkIndex int, peerID string, data []byte) error {
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
	// Requete POST vers d.HTTPFallback + "/report_peer"
	log.Printf("[IMMUNE SYSTEM] Reporting malicious peer: %s", peerID)
}

func (d *Downloader) DownloadChunkFailover(chunkIndex int) error {
	log.Printf("[FAILOVER] Downloading chunk %d directly from LXC Proxy (%s)", chunkIndex, d.HTTPFallback)
	
	// 📊 [SOC] Incrémenter la métrique des failovers P2P -> HTTP
	metrics.Failovers.Inc()
	
	// Code de téléchargement HTTP classique
	return errors.New("not implemented yet")
}
