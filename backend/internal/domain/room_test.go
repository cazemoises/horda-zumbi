package domain

import "testing"

func TestRoom_IsFull(t *testing.T) {
	r := NewRoom("ABCD")
	if r.IsFull() {
		t.Error("empty room should not be full")
	}

	p1, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
	r.Players[p1.ID] = p1
	if r.IsFull() {
		t.Error("room with 1/3 players should not be full")
	}

	p2, _ := NewPlayer(PlayerID("p2"), "Bob", CharacterMedic)
	r.Players[p2.ID] = p2
	if r.IsFull() {
		t.Error("room with 2/3 players should not be full")
	}

	p3, _ := NewPlayer(PlayerID("p3"), "Carol", CharacterScout)
	r.Players[p3.ID] = p3
	if !r.IsFull() {
		t.Error("room with 3/3 players (MaxPlayersPerRoom) should be full")
	}
}

func TestRoom_AddPlayer_Success(t *testing.T) {
	r := NewRoom("ABCD")
	p, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)

	if err := r.AddPlayer(p); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if r.Players[p.ID] != p {
		t.Error("player was not added to the room")
	}
}

func TestRoom_AddPlayer_RejectsFullRoom(t *testing.T) {
	r := NewRoom("ABCD")
	p1, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
	p2, _ := NewPlayer(PlayerID("p2"), "Bob", CharacterMedic)
	p3, _ := NewPlayer(PlayerID("p3"), "Carol", CharacterScout)
	for _, p := range []*Player{p1, p2, p3} {
		if err := r.AddPlayer(p); err != nil {
			t.Fatalf("setup: expected no error adding %v, got %v", p.ID, err)
		}
	}

	p4, _ := NewPlayer(PlayerID("p4"), "Dave", CharacterTank)
	if err := r.AddPlayer(p4); err == nil {
		t.Error("expected error adding player to a full room, got nil")
	}
}

func TestRoom_AddPlayer_RejectsDuplicateID(t *testing.T) {
	r := NewRoom("ABCD")
	p1, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
	if err := r.AddPlayer(p1); err != nil {
		t.Fatalf("setup: expected no error, got %v", err)
	}

	dup, _ := NewPlayer(PlayerID("p1"), "Alice2", CharacterMedic)
	if err := r.AddPlayer(dup); err == nil {
		t.Error("expected error adding player with duplicate ID, got nil")
	}
}
