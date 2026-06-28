package rpc

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

// WsEvent définit la structure des messages envoyés au client Flutter/Dart
type WsEvent struct {
	Type string `json:"type"`
	Data interface{} `json:"data"`
}

type WsServer struct {
	mu          sync.Mutex
	connections []http.ResponseWriter
}

func NewWsServer() *WsServer {
	return &WsServer{}
}

// SendEvent envoie un événement en temps réel à l'UI Dart (sans bloquer)
func (s *WsServer) SendEvent(eventType string, data interface{}) {
	event := WsEvent{
		Type: eventType,
		Data: data,
	}
	b, err := json.Marshal(event)
	if err != nil {
		log.Println("Error marshalling event:", err)
		return
	}
	// Logique de diffusion WebSocket simulée ici.
	// Dans la vraie implémentation, on utilisera github.com/gorilla/websocket
	log.Printf("[RPC -> UI] %s", string(b))
}

// ServeHTTP lance un endpoint WebSocket basique ou SSE
func (s *WsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Implémentation WebSocket/SSE (Server-Sent Events) à finaliser.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"connected"}`))
}
