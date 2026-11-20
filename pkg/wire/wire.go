// Package wire : trames échangées entre pairs et VPS sur une WebSocket (via Cloudflare).
//
// Trame binaire = [type 1o][longueur en-tête 2o][en-tête JSON][données].
package wire

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"t2c/pkg/trust"
)

type Type byte

const (
	TChallenge Type = iota + 1 // VPS → pair : nonce à signer
	THello                     // pair → VPS : identité + PoW + signature du nonce
	TWelcome                   // VPS → pair : identité acceptée
	TError                     // erreur (Q = requête concernée)
	TAnnounce                  // pair → VPS : tickets d'hébergement
	TFind                      // pair → VPS : qui héberge Root ?
	TFound                     // VPS → pair : liste de bénévoles
	TGet                       // pair → VPS : catalog | hosts | manifest | chunk (origine)
	TData                      // réponse à TGet (données dans la charge utile)
	TFwd                       // message opaque (chiffré de bout en bout) vers/depuis un pair
	TAudit                     // VPS → bénévole : défi de stockage
	TAuditResp                 // bénévole → VPS : réponse
	TReport                    // pair → VPS : morceau faux reçu de Peer
	TVolunteer                 // pair → VPS : demande de ticket pour Site
	TTicket                    // VPS → pair : ticket signé par l'origine

	// Fédération : trames échangées uniquement entre relais (jamais avec un pair).
	TMeshHello   // poignée de main mutuelle (clé de relais + signature du nonce)
	TMeshIndex   // annuaire des bénévoles du relais (remplace le précédent)
	TMeshRPC     // appel à l'origine pour le compte d'un relais sans accès direct
	TMeshRPCResp // réponse à TMeshRPC

	// Circuits (routage en oignon) : pair ↔ relais d'entrée, et relais ↔ relais.
	TCircCreate  // ouvre un circuit sur ce lien (Circ) : échange de clés avec le relais
	TCircCreated // réponse du relais à TCircCreate
	TCircCell    // cellule chiffrée en couches, dans un sens ou dans l'autre
	TCircDestroy // ferme le circuit
)

const (
	MaxHeader  = 16 << 10
	MaxPayload = trust.ChunkSize + 4096
	MaxMessage = 3 + MaxHeader + MaxPayload
)

// MeshSubprotocol : liaison entre relais (point d'entrée /v1/mesh).
const MeshSubprotocol = "t2c.mesh.v1"

// MeshPayload : ce qu'un relais signe pour prouver sa clé à un autre relais
// (celui qui ouvre la liaison). Voir HandshakePayload.
func MeshPayload(nonce []byte) []byte { return HandshakePayload(HSMesh, nonce) }

// Rôles de la poignée de main entre relais. Chaque rôle signe dans son propre
// domaine : une signature obtenue dans un rôle (réponse à une liaison, sonde
// d'une adresse, liaison entre bénévoles) ne peut jamais être rejouée dans un
// autre. Sans cela, un faux bénévole pourrait faire signer à un officiel le
// défi d'un AUTRE officiel et s'y présenter sous son identité.
const (
	HSMesh       = "mesh"        // ouverture d'une liaison vers un relais
	HSMeshReply  = "mesh-reply"  // réponse du relais qui accepte
	HSProbe      = "probe"       // sonde d'une adresse publique par un officiel
	HSProbeReply = "probe-reply" // réponse du bénévole sondé
	HSPeer       = "peer"        // liaison entre deux bénévoles publics
	HSPeerReply  = "peer-reply"
)

// HandshakePayload : ce qui est signé pour un rôle donné de la poignée de main.
func HandshakePayload(role string, nonce []byte) []byte {
	return append([]byte("t2c/"+role+"/v1\n"), nonce...)
}

// Subprotocol : sous-protocole WebSocket exigé par le relais. Toute connexion
// qui ne l'annonce pas est refusée avant même l'ouverture de la WebSocket.
const Subprotocol = "t2c.v2"

// HelloPayload : ce que le pair signe pour prouver qu'il détient sa clé.
func HelloPayload(nonce []byte) []byte { return append([]byte("t2c/hello/v1\n"), nonce...) }

// Kinds pour TGet.
const (
	KindCatalog  = "catalog"
	KindHosts    = "hosts"
	KindManifest = "manifest"
	KindChunk    = "chunk"
	// KindVolunteers : relais bénévoles cachés connectés au relais (liste signée par lui)
	KindVolunteers = "volunteers"
)

// HostInfo : un bénévole, tel que le VPS le présente (aucune IP).
type HostInfo struct {
	Peer     string `json:"peer"`
	Pub      []byte `json:"pub"`       // Ed25519
	NoisePub []byte `json:"noise_pub"` // X25519
	NoiseSig []byte `json:"noise_sig"` // signature Ed25519 de NoisePub
	Score    int    `json:"score"`
}

// Header : champs utilisés selon le type (les autres restent vides).
type Header struct {
	Q        uint64         `json:"q,omitempty"` // identifiant de requête
	Nonce    []byte         `json:"nonce,omitempty"`
	Pub      []byte         `json:"pub,omitempty"`
	Mint     uint64         `json:"mint,omitempty"`
	MintBits int            `json:"mint_bits,omitempty"`
	NoisePub []byte         `json:"noise_pub,omitempty"`
	NoiseSig []byte         `json:"noise_sig,omitempty"`
	Sig      []byte         `json:"sig,omitempty"`
	ID       string         `json:"id,omitempty"`
	Err      string         `json:"err,omitempty"`
	Tickets  []trust.Ticket `json:"tickets,omitempty"`
	Ticket   *trust.Ticket  `json:"ticket,omitempty"`
	Root     string         `json:"root,omitempty"`
	Site     string         `json:"site,omitempty"`
	Index    int            `json:"index,omitempty"`
	Kind     string         `json:"kind,omitempty"`
	Hosts    []HostInfo     `json:"hosts,omitempty"`
	To       string         `json:"to,omitempty"`
	From     string         `json:"from,omitempty"`
	SID      uint64         `json:"sid,omitempty"`  // session de bout en bout
	Circ     uint64         `json:"circ,omitempty"` // circuit, propre à chaque lien
	Peer     string         `json:"peer,omitempty"`
	Hash     []byte         `json:"hash,omitempty"`
	// Addr : adresse publique (ws:// ou wss://…/v1/ws) annoncée par un relais
	// bénévole qui accepte d'être relais d'entrée ; vérifiée par sonde.
	Addr string `json:"addr,omitempty"`
	// Version : version de l'application (THello) ou du relais (TMeshHello),
	// comparée au minimum de l'avis de version signé par la racine. Absente
	// chez les applications d'avant l'avis (traitées comme trop anciennes dès
	// qu'un minimum est fixé).
	Version string `json:"version,omitempty"`
}

// ErrObsolete : motif du refus d'une version sous le minimum (TError, Data =
// avis de version signé par la racine, pour afficher la page de mise à jour).
const ErrObsolete = "version obsolète"

// Frame : une trame décodée.
type Frame struct {
	T    Type
	H    Header
	Data []byte
}

func Encode(f *Frame) ([]byte, error) {
	h, err := json.Marshal(&f.H)
	if err != nil {
		return nil, err
	}
	if len(h) > MaxHeader || len(f.Data) > MaxPayload {
		return nil, errors.New("wire: trame trop grande")
	}
	out := make([]byte, 3+len(h)+len(f.Data))
	out[0] = byte(f.T)
	binary.BigEndian.PutUint16(out[1:3], uint16(len(h)))
	copy(out[3:], h)
	copy(out[3+len(h):], f.Data)
	return out, nil
}

func Decode(b []byte) (*Frame, error) {
	if len(b) < 3 {
		return nil, errors.New("wire: trame courte")
	}
	hl := int(binary.BigEndian.Uint16(b[1:3]))
	if hl > MaxHeader || 3+hl > len(b) || len(b)-3-hl > MaxPayload {
		return nil, errors.New("wire: longueurs invalides")
	}
	f := &Frame{T: Type(b[0])}
	if f.T < TChallenge || f.T > TCircDestroy {
		return nil, fmt.Errorf("wire: type inconnu %d", b[0])
	}
	if err := json.Unmarshal(b[3:3+hl], &f.H); err != nil {
		return nil, err
	}
	f.Data = b[3+hl:]
	return f, nil
}

// Conn : WebSocket avec écriture sérialisée et lecture bornée.
type Conn struct {
	ws *websocket.Conn
	wm sync.Mutex
}

const (
	writeWait  = 15 * time.Second
	PongWait   = 75 * time.Second // < 100 s d'inactivité tolérés par Cloudflare
	PingPeriod = 30 * time.Second
)

func NewConn(ws *websocket.Conn) *Conn {
	ws.SetReadLimit(MaxMessage)
	_ = ws.SetReadDeadline(time.Now().Add(PongWait))
	ws.SetPongHandler(func(string) error { return ws.SetReadDeadline(time.Now().Add(PongWait)) })
	return &Conn{ws: ws}
}

func (c *Conn) Send(f *Frame) error {
	b, err := Encode(f)
	if err != nil {
		return err
	}
	c.wm.Lock()
	defer c.wm.Unlock()
	_ = c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(websocket.BinaryMessage, b)
}

func (c *Conn) Ping() error {
	c.wm.Lock()
	defer c.wm.Unlock()
	return c.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait))
}

func (c *Conn) Recv() (*Frame, error) {
	for {
		mt, b, err := c.ws.ReadMessage()
		if err != nil {
			return nil, err
		}
		_ = c.ws.SetReadDeadline(time.Now().Add(PongWait))
		if mt != websocket.BinaryMessage {
			continue
		}
		return Decode(b)
	}
}

func (c *Conn) Close() error { return c.ws.Close() }

// SetReadDeadline : utilisé pendant la poignée de main.
func (c *Conn) SetReadDeadline(t time.Time) error { return c.ws.SetReadDeadline(t) }
