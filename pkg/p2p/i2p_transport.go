package p2p

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// SAMSession représente une connexion active vers le pont SAM d'I2P
type SAMSession struct {
	conn       net.Conn
	B32Address string // Notre adresse intraçable en .b32.i2p
	DestKeys   string // Clé de destination complète
}

// ConnectSAM initie une session SAM v3 standard sur le routeur I2P local (par défaut 127.0.0.1:7656)
func ConnectSAM(samAddr string) (*SAMSession, error) {
	conn, err := net.DialTimeout("tcp", samAddr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("impossible de se connecter au pont SAM I2P sur %s : %w", samAddr, err)
	}

	reader := bufio.NewReader(conn)

	// 1. Handshake HELLO
	fmt.Fprintf(conn, "HELLO VERSION MIN=3.0 MAX=3.3\n")
	resp, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, err
	}
	if !strings.Contains(resp, "RESULT=OK") {
		conn.Close()
		return nil, fmt.Errorf("échec du handshake SAM : %s", strings.TrimSpace(resp))
	}

	// 2. Création de session STREAM transitoire
	fmt.Fprintf(conn, "SESSION CREATE STYLE=STREAM ID=t2c-session DESTINATION=TRANSIENT\n")
	resp, err = reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, err
	}
	if !strings.Contains(resp, "RESULT=OK") {
		conn.Close()
		return nil, fmt.Errorf("échec de création de la session SAM : %s", strings.TrimSpace(resp))
	}

	// Extraction des clés de destination
	var destKeys string
	parts := strings.Split(resp, " ")
	for _, p := range parts {
		if strings.HasPrefix(p, "DESTINATION=") {
			destKeys = strings.TrimPrefix(p, "DESTINATION=")
			destKeys = strings.TrimSpace(destKeys)
		}
	}

	// 3. Résolution de notre propre adresse .b32.i2p
	// Note : Les commandes de contrôle comme NAMING LOOKUP nécessitent une connexion séparée
	lookupConn, err := net.DialTimeout("tcp", samAddr, 5*time.Second)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("échec de connexion lookup : %w", err)
	}
	defer lookupConn.Close()

	lookupReader := bufio.NewReader(lookupConn)
	fmt.Fprintf(lookupConn, "HELLO VERSION MIN=3.0 MAX=3.3\n")
	_, _ = lookupReader.ReadString('\n')

	fmt.Fprintf(lookupConn, "NAMING LOOKUP NAME=ME\n")
	resp, err = lookupReader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, err
	}

	var b32 string
	if strings.Contains(resp, "VALUE=") {
		parts := strings.Split(resp, " ")
		for _, p := range parts {
			if strings.HasPrefix(p, "VALUE=") {
				b32 = strings.TrimPrefix(p, "VALUE=")
				b32 = strings.TrimSpace(b32)
			}
		}
	}

	return &SAMSession{
		conn:       conn,
		B32Address: b32,
		DestKeys:   destKeys,
	}, nil
}

// Close ferme proprement la session I2P
func (s *SAMSession) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}

// TestSAMBridge est un utilitaire rapide pour vérifier la présence du pont SAM
func TestSAMBridge(samAddr string) bool {
	conn, err := net.DialTimeout("tcp", samAddr, 2*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	fmt.Fprintf(conn, "HELLO VERSION MIN=3.0 MAX=3.3\n")
	resp, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	return strings.Contains(resp, "RESULT=OK")
}
