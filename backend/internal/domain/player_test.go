package domain

import "testing"

func TestNewPlayer_ValidInput(t *testing.T) {
	p, err := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if p.Name != "Alice" || p.Character != CharacterTank {
		t.Errorf("player fields not set correctly: %+v", p)
	}
}

func TestNewPlayer_RejectsEmptyName(t *testing.T) {
	cases := []string{"", "   ", "\t\n"}
	for _, name := range cases {
		if _, err := NewPlayer(PlayerID("p1"), name, CharacterTank); err == nil {
			t.Errorf("NewPlayer with name %q: expected error, got nil", name)
		}
	}
}

func TestNewPlayer_RejectsInvalidCharacter(t *testing.T) {
	if _, err := NewPlayer(PlayerID("p1"), "Alice", CharacterID("wizard")); err == nil {
		t.Error("expected error for invalid character ID, got nil")
	}
}
