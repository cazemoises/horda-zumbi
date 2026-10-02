package domain

import (
	"errors"
	"log"
	"math"
	"math/rand"
	"strconv"
	"time"
)

// ErrRoomFull is returned when attempting to add a player to a full room.
var ErrRoomFull = errors.New("room is full")

// ErrPlayerAlreadyInRoom is returned when a player with the same ID already exists in the room.
var ErrPlayerAlreadyInRoom = errors.New("player already in room")

// RoomState is the lifecycle stage of a room.
type RoomState string

const (
	RoomWaiting RoomState = "waiting"
	RoomPlaying RoomState = "playing"
	RoomEnded   RoomState = "ended"
)

// MaxPlayersPerRoom is the maximum number of concurrent players in a room.
const MaxPlayersPerRoom = 3
const ZombieSpawnIntervalTicks = 180
const MaxActiveZombies = 50
const zombieAttackRange = 150.0
const zombieAttackDamage = 25
const zombieContactAttackRange = 40.0
const zombieContactAttackDamage = 10
const zombieContactAttackCooldown = time.Second

// Room groups the players and zombies sharing one match.
type Room struct {
	Code         string
	Players      map[PlayerID]*Player
	Zombies      map[ZombieID]*Zombie
	Wave         int
	State        RoomState
	ticks        int
	nextZombieID int
	collisionMap CollisionMap
}

// NewRoom creates an empty room waiting for players.
func NewRoom(code string) *Room {
	return &Room{
		Code:    code,
		Players: make(map[PlayerID]*Player),
		Zombies: make(map[ZombieID]*Zombie),
		State:   RoomWaiting,
	}
}

// IsFull reports whether the room has reached MaxPlayersPerRoom.
func (r *Room) IsFull() bool {
	return len(r.Players) >= MaxPlayersPerRoom
}

// AddPlayer adds p to the room, rejecting full rooms and duplicate IDs.
func (r *Room) AddPlayer(p *Player) error {
	if r.IsFull() {
		return ErrRoomFull
	}
	if _, exists := r.Players[p.ID]; exists {
		return ErrPlayerAlreadyInRoom
	}

	r.Players[p.ID] = p
	return nil
}

// SetCollisionMap configures the map used by player and zombie movement.
func (r *Room) SetCollisionMap(collisionMap CollisionMap) {
	r.collisionMap = collisionMap
}

// RemovePlayer removes the player with the given ID from the room, if present.
// Removing an ID that is not in the room is a no-op.
func (r *Room) RemovePlayer(id PlayerID) {
	delete(r.Players, id)
}

// SpawnZombie creates and adds a zombie at the given position.
func (r *Room) SpawnZombie(id ZombieID, x, y, speed float64) (*Zombie, error) {
	if math.IsNaN(speed) || math.IsInf(speed, 0) || speed < 0 {
		return nil, errors.New("zombie speed must be finite and non-negative")
	}
	if _, exists := r.Zombies[id]; exists {
		return nil, errors.New("zombie already in room")
	}
	zombie, err := NewZombie(id, x, y)
	if err != nil {
		return nil, err
	}
	zombie.Speed = speed
	r.Zombies[id] = zombie
	return zombie, nil
}

// Tick moves every active zombie toward its nearest player.
func (r *Room) Tick() bool {
	return r.tickAt(time.Now())
}

func (r *Room) tickAt(now time.Time) bool {
	changed := false
	r.ticks++
	for id, zombie := range r.Zombies {
		if zombie.shouldRemove(now) {
			delete(r.Zombies, id)
			changed = true
		}
	}
	if r.ticks%ZombieSpawnIntervalTicks == 0 && len(r.Zombies) < MaxActiveZombies {
		r.Wave++
		for i := 0; i < r.Wave && len(r.Zombies) < MaxActiveZombies; i++ {
			if !r.spawnWaveZombie() {
				break
			}
			changed = true
		}
	}
	for _, zombie := range r.Zombies {
		if zombie.updateAttackState(now) {
			changed = true
		}
		if zombie.Dead {
			continue
		}
		target := nearestPlayer(zombie, r.Players)
		if target != nil && zombie.MoveTowards(target, r.collisionMap) {
			changed = true
		}
		if r.applyContactDamage(zombie, now) {
			changed = true
		}
	}
	return changed
}

func (r *Room) applyContactDamage(zombie *Zombie, now time.Time) bool {
	if !zombie.canAttack(now) {
		return false
	}
	for _, player := range r.Players {
		if math.Hypot(zombie.X-player.X, zombie.Y-player.Y) > zombieContactAttackRange {
			continue
		}
		player.TakeDamage(zombieContactAttackDamage)
		zombie.startAttack(now)
		return true
	}
	return false
}

func (r *Room) spawnWaveZombie() bool {
	var position [2]float64
	if rand.Intn(2) == 0 {
		position = [2]float64{400, float64(rand.Intn(601) - 300)}
		if rand.Intn(2) == 0 {
			position[0] = -400
		}
	} else {
		position = [2]float64{float64(rand.Intn(801) - 400), 300}
		if rand.Intn(2) == 0 {
			position[1] = -300
		}
	}
	for _, player := range r.Players {
		if player.X == position[0] && player.Y == position[1] {
			position[0] += 1
			position[1] += 1
			break
		}
	}

	for {
		r.nextZombieID++
		id := ZombieID("z" + strconv.Itoa(r.nextZombieID))
		if _, exists := r.Zombies[id]; exists {
			continue
		}
		_, err := r.SpawnZombie(id, position[0], position[1], 1)
		return err == nil
	}
}

type attackCandidate struct {
	id       ZombieID
	zombie   *Zombie
	distance float64
}

// Attack resolves the player's directional melee hitbox against active zombies.
func (r *Room) Attack(id PlayerID) int {
	return r.attackAt(id, time.Now())
}

func (r *Room) attackAt(id PlayerID, now time.Time) int {
	player, ok := r.Players[id]
	if !ok {
		return 0
	}
	hitCount := 0
	candidates := make([]attackCandidate, 0)
	for zombieID, zombie := range r.Zombies {
		if zombie.Dead {
			continue
		}
		distance := math.Hypot(zombie.X-player.X, zombie.Y-player.Y)
		samePosition := zombie.X == player.X && zombie.Y == player.Y
		roundedPosition := math.Round(zombie.X) == math.Round(player.X) &&
			math.Round(zombie.Y) == math.Round(player.Y)
		if !samePosition && !roundedPosition && distance > zombieAttackRange {
			continue
		}
		candidates = append(candidates, attackCandidate{
			id:       zombieID,
			zombie:   zombie,
			distance: distance,
		})
	}
	log.Printf("attack candidates: player=%s count=%d distances=%v", player.ID, len(candidates), candidateDistances(candidates))
	for _, candidate := range candidates {
		zombie := candidate.zombie
		healthBefore := zombie.Health
		zombie.TakeDamage(zombieAttackDamage)
		zombie.startHurt(now)
		log.Printf("attack damage: zombie=%s ptr=%p hp_before=%d hp_after=%d", zombie.ID, zombie, healthBefore, zombie.Health)
		hitCount++
		if zombie.IsDead() {
			player.Kills++
			zombie.markDead(now)
		}
	}
	return hitCount
}

func candidateDistances(candidates []attackCandidate) []float64 {
	distances := make([]float64, 0, len(candidates))
	for _, candidate := range candidates {
		distances = append(distances, candidate.distance)
	}
	return distances
}

func nearestPlayer(zombie *Zombie, players map[PlayerID]*Player) *Player {
	var nearest *Player
	nearestDistance := math.Inf(1)
	for _, player := range players {
		distance := math.Hypot(player.X-zombie.X, player.Y-zombie.Y)
		if distance < nearestDistance {
			nearest = player
			nearestDistance = distance
		}
	}
	return nearest
}
