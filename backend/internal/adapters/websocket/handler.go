package websocket

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/coder/websocket"

	"horda-zumbi/backend/internal/ports"
)

// Handler upgrades HTTP requests on /ws/rooms/{code} into websocket
// connections and relays messages between clients of the same room via Hub.
type Handler struct {
	hub *Hub
}

// NewHandler creates a Handler backed by hub.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	if code == "" {
		http.Error(w, "room code is required", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Isso desliga a trava de CORS para testes locais
	})
	if err != nil {
		log.Printf("websocket accept failed for room %s: %v", code, err)
		return
	}

	id := ports.ClientID(newClientID())
	ctx := r.Context()

	_, raw, err := conn.Read(ctx)
	if err != nil {
		conn.Close(websocket.StatusPolicyViolation, "expected join message")
		return
	}

	var join joinMessage
	if err := json.Unmarshal(raw, &join); err != nil || join.Type != "join" {
		writeJSON(conn, errorMessage{Type: "error", Message: "expected a join message"})
		conn.Close(websocket.StatusPolicyViolation, "invalid join message")
		return
	}

	if err := h.hub.Join(code, id, conn, join.Name, join.Character); err != nil {
		writeJSON(conn, errorMessage{Type: "error", Message: err.Error()})
		conn.Close(websocket.StatusPolicyViolation, "join rejected")
		return
	}
	defer h.hub.Leave(code, id)

	for {
		_, message, err := conn.Read(ctx)
		if err != nil {
			conn.Close(websocket.StatusNormalClosure, "")
			return
		}
		var move moveMessage
		if err := json.Unmarshal(message, &move); err == nil && move.Type == "move" {
			h.hub.Move(code, id, move.DX, move.DY)
			continue
		}
		h.hub.Broadcast(code, id, message)
	}
}
