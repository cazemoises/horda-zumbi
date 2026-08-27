package ports

// ClientID identifies one connected websocket client within a room.
type ClientID string

// Broadcaster relays a message from one client to the other clients
// connected to the same room. It carries no game-specific meaning -
// message bytes are passed through as-is.
type Broadcaster interface {
	Broadcast(roomCode string, sender ClientID, message []byte)
}
