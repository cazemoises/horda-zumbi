package domain

import (
	"errors"
	"log"
	"strings"
)

// PlayerID uniquely identifies a player within a room.
type PlayerID string

// Player is a participant connected to a room.
type Player struct {
	ID         PlayerID
	Name       string
	X, Y       float64
	Health     int
	Kills      int
	Character  CharacterID
	DirectionX int
	DirectionY int
}

// ErrEmptyName is returned when a player name is empty or only whitespace.
var ErrEmptyName = errors.New("player name must not be empty")

// ErrInvalidCharacter is returned when a character ID is not one of the fixed playable characters.
var ErrInvalidCharacter = errors.New("invalid character id")

// NewPlayer creates a player at the origin with full health.
func NewPlayer(id PlayerID, name string, character CharacterID) (*Player, error) {
	if strings.TrimSpace(name) == "" {
		return nil, ErrEmptyName
	}

	if !IsValidCharacterID(character) {
		return nil, ErrInvalidCharacter
	}
	return &Player{
		ID:         id,
		Name:       name,
		Health:     100,
		Character:  character,
		DirectionY: 1,
	}, nil
}

// Move updates the player's position by a unit movement vector.
func (p *Player) Move(dx, dy int, maps ...CollisionMap) {
	if dx > 1 {
		dx = 1
	} else if dx < -1 {
		dx = -1
	}
	if dy > 1 {
		dy = 1
	} else if dy < -1 {
		dy = -1
	}
	collisionMap := CollisionMap{}
	if len(maps) > 0 {
		collisionMap = maps[0]
	}
	candidateX, candidateY := p.X+float64(dx), p.Y+float64(dy)
	xTileCol, xTileRow := worldToTile(candidateX, p.Y)
	yTileCol, yTileRow := worldToTile(p.X, candidateY)
	xBlocked := collisionMap.blockedAt(candidateX, p.Y)
	yBlocked := collisionMap.blockedAt(p.X, candidateY)
	log.Printf(
		"player move: before=(%.2f,%.2f) candidateX=(%.2f,%.2f) tileX=(%d,%d) blockedX=%t candidateY=(%.2f,%.2f) tileY=(%d,%d) blockedY=%t",
		p.X, p.Y, candidateX, p.Y, xTileCol, xTileRow, xBlocked,
		p.X, candidateY, yTileCol, yTileRow, yBlocked,
	)
	if !xBlocked {
		p.X += float64(dx)
	}
	if !yBlocked {
		p.Y += float64(dy)
	}
	if dx != 0 || dy != 0 {
		p.DirectionX = dx
		p.DirectionY = dy
	}
}

// TakeDamage reduces the player's health by amount, clamped at 0.
// Non-positive amounts are ignored.
func (p *Player) TakeDamage(amount int) {
	if amount <= 0 {
		return
	}
	p.Health -= amount
	if p.Health < 0 {
		p.Health = 0
	}
}

// IsDead reports whether the player's health has reached 0.
func (p *Player) IsDead() bool {
	return p.Health <= 0
}
