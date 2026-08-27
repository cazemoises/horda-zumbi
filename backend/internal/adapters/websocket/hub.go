// Package websocket is the network foundation for multiplayer rooms: it
// accepts websocket connections at /ws/rooms/{code} and relays messages
// between the clients of the same room. It carries no game logic - message
// bytes are passed through as-is between clients.
package websocket

import (
	"context"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/coder/websocket"

	"horda-zumbi/backend/internal/domain"
	"horda-zumbi/backend/internal/ports"
)

const writeTimeout = 5 * time.Second
const tickInterval = 33 * time.Millisecond

// Hub tracks which clients are connected to which rooms and relays
// messages between them. It implements ports.Broadcaster.
type Hub struct {
	mu       sync.RWMutex
	rooms    map[string]map[ports.ClientID]*websocket.Conn
	roomLoop map[string]chan struct{}
	versions map[string]uint64
	roomRepo ports.RoomRepository
}

// NewHub creates a Hub backed by the given room repository.
func NewHub(roomRepo ports.RoomRepository) *Hub {
	return &Hub{
		rooms:    make(map[string]map[ports.ClientID]*websocket.Conn),
		roomLoop: make(map[string]chan struct{}),
		versions: make(map[string]uint64),
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
	if _, running := h.roomLoop[code]; !running {
		stop := make(chan struct{})
		h.roomLoop[code] = stop
		go h.runGameLoop(code, stop)
	}

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

// Leave removes client id from the room identified by code and its
// domain.Player from the domain.Room, then notifies the remaining clients
// with a "player_left" event. If the room becomes empty, it is dropped from
// both the hub and the room repository - there is no persistence across
// restarts or empty rooms.
func (h *Hub) Leave(code string, id ports.ClientID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.rooms[code]
	if !ok {
		return
	}

	delete(clients, id)

	if room, ok := h.roomRepo.Get(code); ok {
		room.RemovePlayer(domain.PlayerID(id))
	}

	if len(clients) > 0 {
		if payload, err := json.Marshal(playerLeftMessage{Type: "player_left", PlayerID: string(id)}); err == nil {
			for _, c := range clients {
				writeRaw(c, payload)
			}
		}
	}

	if len(clients) == 0 {
		delete(h.rooms, code)
		if stop, ok := h.roomLoop[code]; ok {
			close(stop)
			delete(h.roomLoop, code)
		}
		delete(h.versions, code)
		h.roomRepo.Delete(code)
	}
}

func (h *Hub) runGameLoop(code string, stop <-chan struct{}) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	var previousVersion uint64
	for {
		select {
		case <-ticker.C:
			payload, clients, version := h.stateUpdate(code)
			if payload == nil || version == previousVersion {
				continue
			}
			previousVersion = version
			for _, conn := range clients {
				writeRaw(conn, payload)
			}
		case <-stop:
			return
		}
	}
}

func (h *Hub) stateUpdate(code string) ([]byte, []*websocket.Conn, uint64) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	room, ok := h.roomRepo.Get(code)
	if !ok || len(h.rooms[code]) == 0 {
		return nil, nil, 0
	}
	players := make([]playerView, 0, len(room.Players))
	for _, player := range room.Players {
		players = append(players, newPlayerView(player))
	}
	sort.Slice(players, func(i, j int) bool { return players[i].ID < players[j].ID })
	payload, err := json.Marshal(stateUpdateMessage{Type: "state_update", Players: players})
	if err != nil {
		return nil, nil, 0
	}
	clients := make([]*websocket.Conn, 0, len(h.rooms[code]))
	for _, conn := range h.rooms[code] {
		clients = append(clients, conn)
	}
	return payload, clients, h.versions[code]
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

// Move applies a movement input to the player's domain state.
func (h *Hub) Move(code string, id ports.ClientID, dx, dy int) {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.roomRepo.Get(code)
	if !ok {
		return
	}
	player, ok := room.Players[domain.PlayerID(id)]
	if !ok {
		return
	}
	player.Move(dx, dy)
	h.versions[code]++
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
