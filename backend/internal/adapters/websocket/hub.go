// Package websocket is the network foundation for multiplayer rooms: it
// accepts websocket connections at /ws/rooms/{code} and relays messages
// between the clients of the same room. It carries no game logic - message
// bytes are passed through as-is between clients.
package websocket

import (
	"context"
	"encoding/json"
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

// Join validates name and character, creates the corresponding domain.Player
// and adds it to the room identified by code (creating the room in the
// repository if it does not exist yet). On success it registers conn as
// client id, sends that client a "joined" confirmation carrying the room
// snapshot, and notifies the other clients already in the room with a
// "player_joined" event. It returns an error without registering anything
// if name/character are invalid or the room is already full.
func (h *Hub) Join(code string, id ports.ClientID, conn *websocket.Conn, name string, character domain.CharacterID) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	room := h.roomRepo.GetOrCreate(code)

	player, err := domain.NewPlayer(domain.PlayerID(id), name, character)
	if err != nil {
		return err
	}
	if err := room.AddPlayer(player); err != nil {
		return err
	}

	clients, ok := h.rooms[code]
	if !ok {
		clients = make(map[ports.ClientID]*websocket.Conn)
		h.rooms[code] = clients
	}
	clients[id] = conn

	writeJSON(conn, joinedMessage{
		Type:     "joined",
		PlayerID: string(id),
		Room:     newRoomView(room),
	})

	if payload, err := json.Marshal(playerJoinedMessage{Type: "player_joined", Player: newPlayerView(player)}); err == nil {
		for cid, c := range clients {
			if cid == id {
				continue
			}
			writeRaw(c, payload)
		}
	}

	return nil
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
		writeRaw(conn, message)
	}
}

// writeRaw sends payload as a text frame to conn, ignoring write errors -
// a client that can no longer receive messages is cleaned up separately
// when its own read loop fails.
func writeRaw(conn *websocket.Conn, payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()
	conn.Write(ctx, websocket.MessageText, payload)
}

// writeJSON marshals v and sends it as a text frame to conn.
func writeJSON(conn *websocket.Conn, v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	writeRaw(conn, payload)
	return nil
}
