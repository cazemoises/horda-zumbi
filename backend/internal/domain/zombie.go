package domain

import (
	"errors"
	"math"
)

// ZombieID uniquely identifies a zombie within a room.
type ZombieID string

// Zombie is a hostile entity that pursues a player.
type Zombie struct {
	ID     ZombieID
	X, Y   float64
	Health int
	Target PlayerID
}

// ErrNonFiniteCoordinate is returned when a zombie position is NaN or infinite.
var ErrNonFiniteCoordinate = errors.New("zombie coordinates must be finite")

// NewZombie creates a zombie at the given position with full health.
func NewZombie(id ZombieID, x, y float64) (*Zombie, error) {
	if !isFinite(x) || !isFinite(y) {
		return nil, ErrNonFiniteCoordinate
	}
	return &Zombie{
		ID:     id,
		X:      x,
		Y:      y,
		Health: 50,
	}, nil
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}

// TakeDamage reduces the zombie's health by amount, clamped at 0.
// Non-positive amounts are ignored.
func (z *Zombie) TakeDamage(amount int) {
	if amount <= 0 {
		return
	}
	z.Health -= amount
	if z.Health < 0 {
		z.Health = 0
	}
}

// IsDead reports whether the zombie's health has reached 0.
func (z *Zombie) IsDead() bool {
	return z.Health <= 0
}
