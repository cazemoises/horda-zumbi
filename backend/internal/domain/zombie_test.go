package domain

import (
	"math"
	"testing"
)

func TestNewZombie_ValidInput(t *testing.T) {
	z, err := NewZombie(ZombieID("z1"), 1.5, -2.5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if z.X != 1.5 || z.Y != -2.5 {
		t.Errorf("zombie position not set correctly: %+v", z)
	}
	if z.Speed <= 0 {
		t.Errorf("zombie speed = %v, want positive speed", z.Speed)
	}
}

func TestNewZombie_RejectsNonFiniteCoordinates(t *testing.T) {
	cases := []struct {
		name string
		x, y float64
	}{
		{"NaN x", math.NaN(), 0},
		{"NaN y", 0, math.NaN()},
		{"+Inf x", math.Inf(1), 0},
		{"-Inf y", 0, math.Inf(-1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewZombie(ZombieID("z1"), tc.x, tc.y); err == nil {
				t.Errorf("NewZombie(%v, %v): expected error, got nil", tc.x, tc.y)
			}
		})
	}
}

func TestZombie_TakeDamage_ReducesHealth(t *testing.T) {
	z, _ := NewZombie(ZombieID("z1"), 0, 0)
	z.TakeDamage(20)
	if z.Health != 30 {
		t.Errorf("Health = %d, want 30", z.Health)
	}
}

func TestZombie_TakeDamage_NeverGoesNegative(t *testing.T) {
	z, _ := NewZombie(ZombieID("z1"), 0, 0)
	z.TakeDamage(1000)
	if z.Health != 0 {
		t.Errorf("Health = %d, want 0", z.Health)
	}
}

func TestZombie_TakeDamage_IgnoresNonPositiveAmount(t *testing.T) {
	z, _ := NewZombie(ZombieID("z1"), 0, 0)
	z.TakeDamage(0)
	z.TakeDamage(-10)
	if z.Health != 50 {
		t.Errorf("Health = %d, want 50 (unchanged)", z.Health)
	}
}

func TestZombie_IsDead(t *testing.T) {
	z, _ := NewZombie(ZombieID("z1"), 0, 0)
	if z.IsDead() {
		t.Error("freshly created zombie should not be dead")
	}
	z.TakeDamage(50)
	if !z.IsDead() {
		t.Error("zombie at 0 health should be dead")
	}
}

func TestRoom_TickMovesZombieTowardNearestPlayer(t *testing.T) {
	room := NewRoom("ROOM1")
	player, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
	player.X, player.Y = 10, 0
	if err := room.AddPlayer(player); err != nil {
		t.Fatal(err)
	}
	if _, err := room.SpawnZombie(ZombieID("z1"), 0, 0, 2); err != nil {
		t.Fatal(err)
	}

	room.Tick()

	zombie := room.Zombies[ZombieID("z1")]
	if zombie.X != 2 || zombie.Y != 0 {
		t.Errorf("zombie position = (%v, %v), want (2, 0)", zombie.X, zombie.Y)
	}
}

func TestRoom_TickTargetsNearestPlayer(t *testing.T) {
	room := NewRoom("ROOM2")
	near, _ := NewPlayer(PlayerID("near"), "Near", CharacterTank)
	near.X, near.Y = 3, 0
	far, _ := NewPlayer(PlayerID("far"), "Far", CharacterMedic)
	far.X, far.Y = 0, 10
	_ = room.AddPlayer(near)
	_ = room.AddPlayer(far)
	_, _ = room.SpawnZombie(ZombieID("z1"), 0, 0, 1)

	room.Tick()

	zombie := room.Zombies[ZombieID("z1")]
	if zombie.X != 1 || zombie.Y != 0 {
		t.Errorf("zombie position = (%v, %v), want (1, 0)", zombie.X, zombie.Y)
	}
	if zombie.Target != near.ID {
		t.Errorf("zombie target = %q, want %q", zombie.Target, near.ID)
	}
}
