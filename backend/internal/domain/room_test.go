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
