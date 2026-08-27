// Package websocket is the network foundation for multiplayer rooms: it
// accepts websocket connections at /ws/rooms/{code} and relays messages
// between the clients of the same room. It carries no game logic - message
// bytes are passed through as-is between clients.
package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"

	"horda-zumbi/backend/internal/domain"
	"horda-zumbi/backend/internal/ports"
)

const writeTimeout = 5 * time.Second

// Hub tracks which clients are connected to which rooms and relays
// messages between them. It implements ports.Broadcaster.
type Hub struct {
	mu       sync.RWMutex
	rooms    map[string]map[ports.ClientID]*websocket.Conn
	roomRepo ports.RoomRepository
}

// NewHub creates a Hub backed by the given room repository.
func NewHub(roomRepo ports.RoomRepository) *Hub {
	return &Hub{
		rooms:    make(map[string]map[ports.ClientID]*websocket.Conn),
		roomRepo: roomRepo,
	}
}

// Join registers conn as client id in the room identified by code, creating
// the room in the repository if it does not exist yet. It returns false
// without registering the client if the room already has
// domain.MaxPlayersPerRoom clients connected.
func (h *Hub) Join(code string, id ports.ClientID, conn *websocket.Conn) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.rooms[code]
	if !ok {
		clients = make(map[ports.ClientID]*websocket.Conn)
		h.rooms[code] = clients
	}

	if len(clients) >= domain.MaxPlayersPerRoom {
		return false
	}

	clients[id] = conn
	h.roomRepo.GetOrCreate(code)
	return true
}

// Leave removes client id from the room identified by code. If the room
// becomes empty, it is dropped from both the hub and the room repository -
// there is no persistence across restarts or empty rooms.
func (h *Hub) Leave(code string, id ports.ClientID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.rooms[code]
	if !ok {
		return
	}

	delete(clients, id)

	if len(clients) == 0 {
		delete(h.rooms, code)
		h.roomRepo.Delete(code)
	}
}

// Broadcast relays message to every client in code's room other than
// sender. Messages are always relayed as text frames.
func (h *Hub) Broadcast(code string, sender ports.ClientID, message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for id, conn := range h.rooms[code] {
		if id == sender {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
		conn.Write(ctx, websocket.MessageText, message)
		cancel()
	}
}
