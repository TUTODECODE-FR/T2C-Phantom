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
)

const (
	MaxHeader  = 16 << 10
	MaxPayload = trust.ChunkSize + 4096
	MaxMessage = 3 + MaxHeader + MaxPayload
)

// HelloPayload : ce que le pair signe pour prouver qu'il détient sa clé.
func HelloPayload(nonce []byte) []byte { return append([]byte("t2c/hello/v1\n"), nonce...) }

// Kinds pour TGet.
const (
	KindCatalog  = "catalog"
	KindHosts    = "hosts"
	KindManifest = "manifest"
	KindChunk    = "chunk"
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
	SID      uint64         `json:"sid,omitempty"` // session de bout en bout
	Peer     string         `json:"peer,omitempty"`
	Hash     []byte         `json:"hash,omitempty"`
}

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
	if f.T < TChallenge || f.T > TTicket {
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
