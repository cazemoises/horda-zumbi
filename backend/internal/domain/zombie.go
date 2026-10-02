package domain

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

// ZombieID uniquely identifies a zombie within a room.
type ZombieID string

// Zombie is a hostile entity that pursues a player.
type Zombie struct {
	ID             ZombieID
	Type           int
	X, Y           float64
	Health         int
	Speed          float64
	Target         PlayerID
	Attacking      bool
	Hurt           bool
	Dead           bool
	nextAttackAt   time.Time
	attackingUntil time.Time
	hurtUntil      time.Time
	deadAt         time.Time
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
		Type:   rand.Intn(4) + 1,
		X:      x,
		Y:      y,
		Health: 50,
		Speed:  1,
	}, nil
}

// MoveTowards advances the zombie toward the player's current position.
func (z *Zombie) MoveTowards(player *Player, maps ...CollisionMap) bool {
	dx := player.X - z.X
	dy := player.Y - z.Y
	distance := math.Hypot(dx, dy)
	z.Target = player.ID
	if distance == 0 || z.Speed <= 0 {
		return false
	}

	distanceMoved := math.Min(z.Speed, distance)
	stepX := dx / distance * distanceMoved
	stepY := dy / distance * distanceMoved
	collisionMap := CollisionMap{}
	if len(maps) > 0 {
		collisionMap = maps[0]
	}
	moved := false
	if !collisionMap.blockedAt(z.X+stepX, z.Y) {
		z.X += stepX
		moved = true
	}
	if !collisionMap.blockedAt(z.X, z.Y+stepY) {
		z.Y += stepY
		moved = true
	}
	return moved
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

func (z *Zombie) startHurt(now time.Time) {
	z.hurtUntil = now.Add(300 * time.Millisecond)
	z.Hurt = true
}

func (z *Zombie) markDead(now time.Time) {
	z.Dead = true
	z.deadAt = now
	z.Hurt = false
}

func (z *Zombie) shouldRemove(now time.Time) bool {
	return z.Dead && !now.Before(z.deadAt.Add(time.Second))
}

func (z *Zombie) canAttack(now time.Time) bool {
	return !now.Before(z.nextAttackAt)
}

func (z *Zombie) startAttack(now time.Time) {
	z.nextAttackAt = now.Add(time.Second)
	z.attackingUntil = now.Add(500 * time.Millisecond)
	z.Attacking = true
}

func (z *Zombie) updateAttackState(now time.Time) bool {
	changed := false
	if z.Attacking && !now.Before(z.attackingUntil) {
		z.Attacking = false
		changed = true
	}
	if z.Hurt && !now.Before(z.hurtUntil) {
		z.Hurt = false
		changed = true
	}
	return changed
}
