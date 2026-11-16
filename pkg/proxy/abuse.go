package proxy

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"t2c/pkg/utils"
)

const (
	abuseWindow     = 5 * time.Minute
	floodWindow     = 10 * time.Second
	floodMax        = 40 // req / floodWindow → DDoS local
	spoofWarnAt     = 3
	spoofSignalAt   = 5
	spoofThrottleAt = 8
	tamperSignalAt  = 3
	maxEventsKept   = 50
)

type abuseEvent struct {
	At      time.Time `json:"at"`
	Kind    string    `json:"kind"` // spoof | tamper | flood | signal
	Client  string    `json:"client"`
	Detail  string    `json:"detail"`
	Strikes int       `json:"strikes"`
}

type clientBucket struct {
	spoof          int
	tamper         int
	hits           []time.Time
	throttledUntil time.Time
	signaled       bool
}

// AbuseWatch : fausses demandes signalées si répétées / modifiées / flood.
// L'accès HTTP aux vrais sites asso reste possible (on ne verrouille pas tout le monde).
type AbuseWatch struct {
	mu      sync.Mutex
	clients map[string]*clientBucket
	events  []abuseEvent
}

var GlobalAbuse = &AbuseWatch{clients: make(map[string]*clientBucket)}

func clientKey(r *http.Request) string {
	// Proxy local : les requêtes viennent du navigateur de la machine.
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if host == "" || host == "127.0.0.1" || host == "::1" {
		return "navigateur-local"
	}
	return host
}

func (a *AbuseWatch) bucket(key string) *clientBucket {
	b, ok := a.clients[key]
	if !ok {
		b = &clientBucket{}
		a.clients[key] = b
	}
	return b
}

func (a *AbuseWatch) push(ev abuseEvent) {
	a.events = append(a.events, ev)
	if len(a.events) > maxEventsKept {
		a.events = a.events[len(a.events)-maxEventsKept:]
	}
}

func (a *AbuseWatch) pruneHits(b *clientBucket, now time.Time) {
	cut := now.Add(-floodWindow)
	i := 0
	for _, t := range b.hits {
		if t.After(cut) {
			b.hits[i] = t
			i++
		}
	}
	b.hits = b.hits[:i]
}

// ObserveRequest anti-DDoS + détection requêtes modifiées. Renvoie false si throttle dur.
// Les GET HTTP vers un vrai .t2c restent autorisés même après signalement spoof.
func (a *AbuseWatch) ObserveRequest(r *http.Request) (allowed bool, reason string) {
	// Bruit navigateur (CONNECT google/mozilla via proxy) : ignorer pour flood/throttle.
	if r.Method == http.MethodConnect {
		h := strings.ToLower(strings.Split(r.Host, ":")[0])
		if !strings.HasSuffix(h, ".t2c") {
			return true, ""
		}
	}

	key := clientKey(r)
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	b := a.bucket(key)
	a.pruneHits(b, now)
	b.hits = append(b.hits, now)

	if now.Before(b.throttledUntil) && r.Method == http.MethodConnect {
		return false, "throttle: trop de fausses/abus CONNECT"
	}

	// Flood / DDoS via le proxy
	if len(b.hits) > floodMax {
		b.throttledUntil = now.Add(2 * time.Minute)
		a.push(abuseEvent{At: now, Kind: "flood", Client: key, Detail: fmt.Sprintf("%d req/%s", len(b.hits), floodWindow), Strikes: len(b.hits)})
		utils.LogSecurityEvent("PROXY_FLOOD", fmt.Sprintf("%s %d hits", key, len(b.hits)), key)
		return false, "ddos: rate limit proxy"
	}

	// Requête modifiée : Host incohérent avec l'URL / CONNECT
	if tamper := detectTamper(r); tamper != "" {
		b.tamper++
		a.push(abuseEvent{At: now, Kind: "tamper", Client: key, Detail: tamper, Strikes: b.tamper})
		if b.tamper >= tamperSignalAt && !b.signaled {
			b.signaled = true
			a.push(abuseEvent{At: now, Kind: "signal", Client: key, Detail: "requêtes modifiées répétées", Strikes: b.tamper})
			utils.LogSecurityEvent("PROXY_TAMPER_SIGNAL", tamper, key)
		}
		// On laisse passer le contenu HTTP légitime ; on refuse seulement CONNECT si déjà throttlé
		if b.tamper >= spoofThrottleAt {
			b.throttledUntil = now.Add(abuseWindow)
			if r.Method == http.MethodConnect {
				return false, "throttle: requêtes modifiées"
			}
		}
	}
	return true, ""
}

func detectTamper(r *http.Request) string {
	reqHost := strings.ToLower(strings.Split(r.Host, ":")[0])
	if r.Method == http.MethodConnect {
		connectHost := strings.ToLower(strings.Split(r.RequestURI, ":")[0])
		if connectHost == "" {
			connectHost = strings.ToLower(strings.Split(r.URL.Host, ":")[0])
		}
		if reqHost != "" && connectHost != "" && reqHost != connectHost {
			return fmt.Sprintf("CONNECT host≠URI (%s vs %s)", reqHost, connectHost)
		}
		return ""
	}
	// Absolute-form proxy : Host doit matcher l'URL
	if r.URL != nil && r.URL.Host != "" {
		urlHost := strings.ToLower(strings.Split(r.URL.Host, ":")[0])
		if reqHost != "" && urlHost != "" && reqHost != urlHost {
			return fmt.Sprintf("Host≠URL (%s vs %s)", reqHost, urlHost)
		}
	}
	return ""
}

// RecordSpoof : fausse demande HTTPS (hors catalogue asso). Site HTTP réel reste accessible.
func (a *AbuseWatch) RecordSpoof(r *http.Request, domain string) (signaled bool, throttled bool) {
	key := clientKey(r)
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	b := a.bucket(key)
	b.spoof++
	detail := "faux CONNECT " + domain
	a.push(abuseEvent{At: now, Kind: "spoof", Client: key, Detail: detail, Strikes: b.spoof})

	if b.spoof >= spoofWarnAt {
		fmt.Println("[attention] abuse spoof:", key, b.spoof, domain)
	}
	if b.spoof >= spoofSignalAt {
		signaled = true
		if !b.signaled || b.spoof == spoofSignalAt {
			b.signaled = true
			a.push(abuseEvent{At: now, Kind: "signal", Client: key, Detail: "spoof HTTPS répété", Strikes: b.spoof})
			utils.LogSecurityEvent("PROXY_SPOOF_SIGNAL", detail, key)
			fmt.Println("SIGNAL spoof HTTPS:", key, "strikes=", b.spoof, domain)
		}
	}
	if b.spoof >= spoofThrottleAt {
		throttled = true
		b.throttledUntil = now.Add(abuseWindow)
	}
	return signaled, throttled
}

// Reset clears throttle/spoof counters (redémarrage logique sans tuer le process).
func (a *AbuseWatch) Reset() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.clients = make(map[string]*clientBucket)
	a.events = nil
}

// Snapshot pour le dashboard.
func (a *AbuseWatch) Snapshot() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	evs := make([]abuseEvent, len(a.events))
	copy(evs, a.events)
	signaled := 0
	throttled := 0
	now := time.Now()
	for _, b := range a.clients {
		if b.signaled {
			signaled++
		}
		if now.Before(b.throttledUntil) {
			throttled++
		}
	}
	return map[string]any{
		"events":    evs,
		"signaled":  signaled,
		"throttled": throttled,
		"note":      "protection locale du proxy (navigateur de cette machine)",
	}
}
