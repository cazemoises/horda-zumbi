package domain

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

// NewPlayer creates a player at the origin with full health.
func NewPlayer(id PlayerID, name string, character CharacterID) *Player {
	return &Player{
		ID:        id,
		Name:      name,
		Health:    100,
		Character: character,
	}
}
