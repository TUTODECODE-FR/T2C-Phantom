package swarm

import (
	"encoding/json"
	"testing"
)

// FuzzDeserializeRecord teste la résilience du parseur JSON de T2C-Phantom
// face à des paquets P2P aléatoires et corrompus. (Prévention Déni de Service / Panic)
func FuzzDeserializeRecord(f *testing.F) {
	// Corpus initial (Seed) - Des exemples de paquets valides et semi-valides
	f.Add([]byte(`{"timestamp":1672531200,"zip_hash":"abc123def456","signature":"c2lnbmF0dXJl"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"timestamp":"invalid_string"}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(``))
	
	f.Fuzz(func(t *testing.T, data []byte) {
		// L'objectif du Fuzzing ici est de s'assurer que json.Unmarshal ou notre code
		// ne fait JAMAIS de panic() lorsqu'il reçoit un byte slice malformé depuis le réseau.
		var r RoutingRecord
		json.Unmarshal(data, &r)
		
		// On peut aussi tester le Deserialize wrapper qui fait la même chose
		DeserializeRecord(data)
	})
}
