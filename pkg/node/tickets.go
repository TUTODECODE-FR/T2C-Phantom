package node

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"t2c/pkg/trust"
)

// Les tickets de bénévolat survivent aux redémarrages : au retour, le pair
// réannonce ses copies sans redemander de ticket.

func (n *Node) ticketsPath() string { return filepath.Join(n.cfg.DataDir, "tickets.json") }

func (n *Node) loadTickets() {
	b, err := os.ReadFile(n.ticketsPath())
	if err != nil {
		return
	}
	var ts []trust.Ticket
	if json.Unmarshal(b, &ts) != nil {
		return
	}
	now := time.Now()
	for _, t := range ts {
		if t.Peer == n.id.ID() && t.Valid(now) && t.Verify(n.cfg.RootKey) == nil {
			n.tickets = append(n.tickets, t)
		}
	}
}

func (n *Node) saveTickets(ts []trust.Ticket) {
	n.mu.Lock()
	n.tickets = ts
	n.mu.Unlock()
	b, err := json.Marshal(ts)
	if err != nil {
		return
	}
	tmp := n.ticketsPath() + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, n.ticketsPath())
	}
}
