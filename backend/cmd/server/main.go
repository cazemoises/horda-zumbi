// Command server wires the horda-zumbi backend's in-memory room repository
// and websocket relay into an HTTP server and starts listening.
package main

import (
	"log"
	"net/http"
	"os"

	"horda-zumbi/backend/internal/adapters/httpserver"
	"horda-zumbi/backend/internal/adapters/memory"
	wsadapter "horda-zumbi/backend/internal/adapters/websocket"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	roomRepo := memory.NewRoomRepository()
	hub := wsadapter.NewHub(roomRepo)
	wsHandler := wsadapter.NewHandler(hub)

	server := httpserver.NewServer(wsHandler)

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, server); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
