// Package httpserver wires the game's HTTP-facing endpoints - the health
// check and the websocket room endpoint - into a single http.Handler.
package httpserver

import (
	"encoding/json"
	"net/http"
)

// NewServer builds the HTTP handler exposing GET /health and the websocket
// room endpoint served by wsHandler.
func NewServer(wsHandler http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.Handle("/ws/rooms/{code}", wsHandler)

	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
