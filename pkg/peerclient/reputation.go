package peerclient

import (
	"sort"
	"sync"

	"t2c/pkg/wire"
)

// Reputation : score local de chaque pair, vu par ce client.
// Morceau valide → +1 ; erreur/lenteur → −3 ; morceau faux → ban définitif.
type Reputation struct {
	mu     sync.Mutex
	score  map[string]int
	banned map[string]bool
}

func NewReputation() *Reputation {
	return &Reputation{score: map[string]int{}, banned: map[string]bool{}}
}

func (r *Reputation) Good(p string) { r.mu.Lock(); r.score[p]++; r.mu.Unlock() }
func (r *Reputation) Bad(p string)  { r.mu.Lock(); r.score[p] -= 3; r.mu.Unlock() }
func (r *Reputation) Ban(p string)  { r.mu.Lock(); r.banned[p] = true; r.mu.Unlock() }

func (r *Reputation) Banned(p string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.banned[p]
}

func (r *Reputation) Score(p string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.score[p]
}

// rank : non bannis, pas trop mal notés localement, triés par (score local + score origine).
func (r *Reputation) rank(hosts []wire.HostInfo) []wire.HostInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := hosts[:0:0]
	for _, h := range hosts {
		if !r.banned[h.Peer] && r.score[h.Peer] > -15 {
			out = append(out, h)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return r.score[out[i].Peer]+out[i].Score > r.score[out[j].Peer]+out[j].Score
	})
	return out
}
