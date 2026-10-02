package domain

import (
	"math"
	"testing"
	"time"
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

func TestRoom_Tick_ZombieStopsOnBlockedAxis(t *testing.T) {
	var collisionMap CollisionMap
	collisionMap.blocked[250][250] = true
	room := NewRoom("COLLISION-ZOMBIE")
	room.SetCollisionMap(collisionMap)
	player, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
	player.X, player.Y = 10, 0
	_ = room.AddPlayer(player)
	zombie, _ := room.SpawnZombie(ZombieID("z1"), 0, 0, 1)

	room.Tick()

	if zombie.X != 0 || zombie.Y != 0 {
		t.Errorf("zombie position = (%v, %v), want (0, 0) with blocked X", zombie.X, zombie.Y)
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

func TestRoom_TickZombieContactDamage(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	t.Run("within range damages player", func(t *testing.T) {
		room := NewRoom("CONTACT1")
		player, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
		_ = room.AddPlayer(player)
		_, _ = room.SpawnZombie(ZombieID("z1"), 0, 0, 0)

		room.tickAt(now)

		if player.Health != 90 {
			t.Errorf("player health = %d, want 90", player.Health)
		}
	})

	t.Run("outside range does not damage player", func(t *testing.T) {
		room := NewRoom("CONTACT2")
		player, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
		_ = room.AddPlayer(player)
		_, _ = room.SpawnZombie(ZombieID("z1"), zombieContactAttackRange+1, 0, 0)

		room.tickAt(now)

		if player.Health != 100 {
			t.Errorf("player health = %d, want 100", player.Health)
		}
	})

	t.Run("cooldown prevents repeated damage", func(t *testing.T) {
		room := NewRoom("CONTACT3")
		player, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
		_ = room.AddPlayer(player)
		_, _ = room.SpawnZombie(ZombieID("z1"), 0, 0, 0)

		room.tickAt(now)
		room.tickAt(now.Add(500 * time.Millisecond))
		if player.Health != 90 {
			t.Errorf("player health after cooldown tick = %d, want 90", player.Health)
		}
		room.tickAt(now.Add(zombieContactAttackCooldown))
		if player.Health != 80 {
			t.Errorf("player health after cooldown = %d, want 80", player.Health)
		}
	})
}

func TestRoom_TickSpawnsZombieAfterSpawnInterval(t *testing.T) {
	room := NewRoom("WAVE1")

	for i := 0; i < ZombieSpawnIntervalTicks; i++ {
		room.Tick()
	}

	if len(room.Zombies) != 1 {
		t.Fatalf("zombies = %d, want 1 after %d ticks", len(room.Zombies), ZombieSpawnIntervalTicks)
	}
	for _, zombie := range room.Zombies {
		if math.Abs(zombie.X) < 300 && math.Abs(zombie.Y) < 200 {
			t.Errorf("zombie spawned away from arena edge at (%v, %v)", zombie.X, zombie.Y)
		}
	}
}

func TestRoom_TickStopsSpawningAtZombieLimit(t *testing.T) {
	room := NewRoom("WAVE2")

	for i := 0; i < ZombieSpawnIntervalTicks*(MaxActiveZombies+2); i++ {
		room.Tick()
	}

	if len(room.Zombies) != MaxActiveZombies {
		t.Errorf("zombies = %d, want capped at %d", len(room.Zombies), MaxActiveZombies)
	}
}

func TestRoom_AttackDamagesAndDelaysZombieRemoval(t *testing.T) {
	room := NewRoom("COMBAT1")
	player, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
	if err := room.AddPlayer(player); err != nil {
		t.Fatal(err)
	}
	_, err := room.SpawnZombie(ZombieID("z1"), 0, 30, 0)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if room.attackAt(player.ID, now) == 0 {
		t.Fatal("expected attack to hit zombie")
	}
	zombie := room.Zombies[ZombieID("z1")]
	if zombie.Health != 25 || !zombie.Hurt {
		t.Errorf("zombie state after hit = health %d, hurt %t; want 25, true", zombie.Health, zombie.Hurt)
	}
	if room.attackAt(player.ID, now.Add(300*time.Millisecond)) == 0 {
		t.Fatal("expected second attack to hit zombie")
	}
	zombie = room.Zombies[ZombieID("z1")]
	if !zombie.Dead || zombie.Health != 0 {
		t.Errorf("zombie state after fatal hit = dead %t, health %d; want true, 0", zombie.Dead, zombie.Health)
	}
	room.tickAt(now.Add(500 * time.Millisecond))
	if _, exists := room.Zombies[ZombieID("z1")]; !exists {
		t.Error("dead zombie should remain during death animation")
	}
	room.tickAt(now.Add(1300 * time.Millisecond))
	if _, exists := room.Zombies[ZombieID("z1")]; exists {
		t.Error("dead zombie should be removed after death animation")
	}
	if player.Kills != 1 {
		t.Errorf("player kills = %d, want 1", player.Kills)
	}
}

func TestRoom_DeadZombieDoesNotMoveOrDamage(t *testing.T) {
	room := NewRoom("COMBAT-DEAD")
	player, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
	_ = room.AddPlayer(player)
	zombie, _ := room.SpawnZombie(ZombieID("z1"), 0, 0, 1)
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	zombie.markDead(now)

	room.tickAt(now.Add(100 * time.Millisecond))

	if player.Health != 100 {
		t.Errorf("player health = %d, want 100", player.Health)
	}
	if zombie.X != 0 || zombie.Y != 0 {
		t.Errorf("dead zombie moved to (%v, %v)", zombie.X, zombie.Y)
	}
	if _, exists := room.Zombies[zombie.ID]; !exists {
		t.Error("dead zombie should remain during death animation")
	}
}

func TestRoom_AttackUsesLastMovementDirection(t *testing.T) {
	room := NewRoom("COMBAT2")
	player, _ := NewPlayer(PlayerID("p1"), "Alice", CharacterTank)
	if err := room.AddPlayer(player); err != nil {
		t.Fatal(err)
	}
	player.Move(1, 0)
	player.Move(0, 0)
	if _, err := room.SpawnZombie(ZombieID("z1"), 60, 0, 0); err != nil {
		t.Fatal(err)
	}

	if room.Attack(player.ID) == 0 {
		t.Fatal("expected attack to use the player's last direction")
	}
}
