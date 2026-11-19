package wire

// LoadReport : charge d'un relais OFFICIEL, envoyée au serveur maison toutes
// les minutes (POST /v1/load, ou RPC « load » via un relais amont). Aucune
// donnée identifiante : des compteurs et, au plus, des clés de relais
// bénévoles qui se comportent mal. Les bénévoles n'en envoient jamais.
type LoadReport struct {
	Relay          string   `json:"relay"` // clé du relais (hex), fixée par celui qui transmet
	At             int64    `json:"at"`
	Period         int64    `json:"period"` // secondes couvertes
	Peers          int      `json:"peers"`
	Circuits       int      `json:"circuits"`
	Volunteers     int      `json:"volunteers"`
	NewVolunteers  int      `json:"new_volunteers"` // admis pendant la période
	BytesPerSec    float64  `json:"bytes_per_sec"`  // débit moyen relayé
	CapBytesPerSec float64  `json:"cap_bytes_per_sec,omitempty"`
	Misbehaving    []string `json:"misbehaving,omitempty"` // bénévoles fautifs (clés hex)
}

// MaxMisbehaving : clés fautives max par rapport.
const MaxMisbehaving = 64
