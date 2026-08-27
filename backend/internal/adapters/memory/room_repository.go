// Package memory implements the game's ports with plain in-process state.
// Nothing here survives a restart - there is no persistence layer.
package memory

import (
	"sync"

	"horda-zumbi/backend/internal/domain"
)

// RoomRepository is a thread-safe, in-memory ports.RoomRepository.
type RoomRepository struct {
	mu    sync.RWMutex
	rooms map[string]*domain.Room
}

// NewRoomRepository creates an empty repository.
func NewRoomRepository() *RoomRepository {
	return &RoomRepository{
		rooms: make(map[string]*domain.Room),
	}
}

func (r *RoomRepository) GetOrCreate(code string) *domain.Room {
	r.mu.Lock()
	defer r.mu.Unlock()

	room, ok := r.rooms[code]
	if !ok {
		room = domain.NewRoom(code)
		r.rooms[code] = room
	}
	return room
}

func (r *RoomRepository) Get(code string) (*domain.Room, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	room, ok := r.rooms[code]
	return room, ok
}

func (r *RoomRepository) Delete(code string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.rooms, code)
}
