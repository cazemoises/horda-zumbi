package domain

// RoomState is the lifecycle stage of a room.
type RoomState string

const (
	RoomWaiting RoomState = "waiting"
	RoomPlaying RoomState = "playing"
	RoomEnded   RoomState = "ended"
)

// MaxPlayersPerRoom is the maximum number of concurrent players in a room.
const MaxPlayersPerRoom = 3

// Room groups the players and zombies sharing one match.
type Room struct {
	Code    string
	Players map[PlayerID]*Player
	Zombies map[ZombieID]*Zombie
	Wave    int
	State   RoomState
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
