// Command server wires the horda-zumbi backend's in-memory room repository
// and websocket relay into an HTTP server and starts listening.
package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"horda-zumbi/backend/internal/adapters/httpserver"
	"horda-zumbi/backend/internal/adapters/memory"
	wsadapter "horda-zumbi/backend/internal/adapters/websocket"
	"horda-zumbi/backend/internal/domain"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	roomRepo := memory.NewRoomRepository()
	mapData, err := loadMap()
	if err != nil {
		log.Fatalf("load collision map: %v", err)
	}
	collisionMap, err := domain.LoadCollisionMap(mapData)
	if err != nil {
		log.Fatalf("parse collision map: %v", err)
	}
	hub := wsadapter.NewHub(roomRepo, collisionMap)
	wsHandler := wsadapter.NewHandler(hub)

	server := httpserver.NewServer(wsHandler)

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, server); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func loadMap() ([]byte, error) {
	paths := []string{
		filepath.Join("frontend", "src", "assets", "tiles", "urban-kit", "map.json"),
		filepath.Join("..", "frontend", "src", "assets", "tiles", "urban-kit", "map.json"),
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
	}
	return nil, os.ErrNotExist
}
