package ports

import "horda-zumbi/backend/internal/domain"

// RoomRepository stores rooms in memory for the lifetime of the process.
// There is no persistence: implementations lose all state on restart.
type RoomRepository interface {
	// GetOrCreate returns the room for code, creating an empty waiting room
	// if it does not exist yet.
	GetOrCreate(code string) *domain.Room

	// Get returns the room for code and whether it exists.
	Get(code string) (*domain.Room, bool)

	// Delete removes a room, e.g. once its last player has left.
	Delete(code string)
}
