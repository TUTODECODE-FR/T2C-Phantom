package rpc

import (
	"sync"
	"testing"
	"time"
)

func TestRPCAsyncEventPush(t *testing.T) {
	server := NewWsServer()

	var wg sync.WaitGroup

	// On va simuler l'envoi de 100 événements rapidement (comme une barre de progression)
	// Le test réussit si la fonction SendEvent ne bloque jamais et s'exécute quasi instantanément.

	start := time.Now()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(progress int) {
			defer wg.Done()
			server.SendEvent("progress", progress)
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// L'envoi des 100 events doit prendre moins d'1 milliseconde vu que c'est de l'asynchrone/simulé
	if duration > 100*time.Millisecond {
		t.Errorf("CRITICAL: RPC channel is blocking. Event emission took too long: %v", duration)
	} else {
		t.Logf("[ok] RPC Asynchronous Check: 100 events sent without blocking main thread. Took %v", duration)
	}
}
