package domain

import (
	"errors"
	"math"
	"strconv"
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
const ZombieSpawnIntervalTicks = 90
const MaxActiveZombies = 50

// Room groups the players and zombies sharing one match.
type Room struct {
	Code         string
	Players      map[PlayerID]*Player
	Zombies      map[ZombieID]*Zombie
	Wave         int
	State        RoomState
	ticks        int
	nextZombieID int
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
	changed := false
	r.ticks++
	if r.ticks%ZombieSpawnIntervalTicks == 0 && len(r.Zombies) < MaxActiveZombies {
		if r.spawnWaveZombie() {
			changed = true
		}
	}
	for _, zombie := range r.Zombies {
		target := nearestPlayer(zombie, r.Players)
		if target != nil && zombie.MoveTowards(target) {
			changed = true
		}
	}
	return changed
}

func (r *Room) spawnWaveZombie() bool {
	positions := [][2]float64{{400, 0}, {-400, 0}, {0, 300}, {0, -300}}
	position := positions[r.nextZombieID%len(positions)]
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
