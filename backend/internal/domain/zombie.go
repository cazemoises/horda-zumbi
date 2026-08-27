package domain

// ZombieID uniquely identifies a zombie within a room.
type ZombieID string

// Zombie is a hostile entity that pursues a player.
type Zombie struct {
	ID     ZombieID
	X, Y   float64
	Health int
	Target PlayerID
}

// NewZombie creates a zombie at the given position with full health.
func NewZombie(id ZombieID, x, y float64) *Zombie {
	return &Zombie{
		ID:     id,
		X:      x,
		Y:      y,
		Health: 50,
	}
}
