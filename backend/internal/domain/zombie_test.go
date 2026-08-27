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
