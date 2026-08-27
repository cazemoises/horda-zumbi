package websocket

import "horda-zumbi/backend/internal/domain"

// joinMessage is the client's mandatory first message after connecting.
type joinMessage struct {
	Type      string             `json:"type"`
	Name      string             `json:"name"`
	Character domain.CharacterID `json:"character"`
}

// playerView is the public representation of a domain.Player sent over the
// wire.
type playerView struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Character domain.CharacterID `json:"character"`
	X         float64            `json:"x"`
	Y         float64            `json:"y"`
	Health    int                `json:"health"`
}

// roomView is the public snapshot of a domain.Room sent to a client that
// just joined it.
type roomView struct {
	Code    string       `json:"code"`
	Wave    int          `json:"wave"`
	State   string       `json:"state"`
	Players []playerView `json:"players"`
}

// joinedMessage confirms a successful join to the client that sent it.
type joinedMessage struct {
	Type     string   `json:"type"`
	PlayerID string   `json:"player_id"`
	Room     roomView `json:"room"`
}

// playerJoinedMessage notifies the other clients already in a room that a
// new player joined it.
type playerJoinedMessage struct {
	Type   string     `json:"type"`
	Player playerView `json:"player"`
}

// playerLeftMessage notifies the other clients in a room that a player
// disconnected.
type playerLeftMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"player_id"`
}

// errorMessage reports a rejected join to the client that sent it.
type errorMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

func newPlayerView(p *domain.Player) playerView {
	return playerView{
		ID:        string(p.ID),
		Name:      p.Name,
		Character: p.Character,
		X:         p.X,
		Y:         p.Y,
		Health:    p.Health,
	}
}

func newRoomView(r *domain.Room) roomView {
	players := make([]playerView, 0, len(r.Players))
	for _, p := range r.Players {
		players = append(players, newPlayerView(p))
	}
	return roomView{
		Code:    r.Code,
		Wave:    r.Wave,
		State:   string(r.State),
		Players: players,
	}
}
