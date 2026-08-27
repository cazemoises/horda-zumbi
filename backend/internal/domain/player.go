package domain

import (
	"errors"
	"strings"
)

// PlayerID uniquely identifies a player within a room.
type PlayerID string

// Player is a participant connected to a room.
type Player struct {
	ID        PlayerID
	Name      string
	X, Y      float64
	Health    int
	Character CharacterID
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
		ID:        id,
		Name:      name,
		Health:    100,
		Character: character,
	}, nil
}
