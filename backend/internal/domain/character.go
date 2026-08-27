package domain

// CharacterID identifies one of the fixed playable characters.
type CharacterID string

const (
	CharacterTank    CharacterID = "tank"
	CharacterMedic   CharacterID = "medic"
	CharacterScout   CharacterID = "scout"
)

// Character describes a playable character and its unique ability.
// Ability behavior is not implemented yet - this is a placeholder for the
// future gameplay layer.
type Character struct {
	ID      CharacterID
	Name    string
	Ability string
}

// Characters lists the fixed set of playable characters available in a room.
var Characters = []Character{
	{ID: CharacterTank, Name: "Tank", Ability: "Reduz dano recebido (placeholder)"},
	{ID: CharacterMedic, Name: "Médico", Ability: "Cura aliados próximos (placeholder)"},
	{ID: CharacterScout, Name: "Batedor", Ability: "Velocidade de movimento aumentada (placeholder)"},
}
