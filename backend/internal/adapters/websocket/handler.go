package websocket

import (
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

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("websocket accept failed for room %s: %v", code, err)
		return
	}

	id := ports.ClientID(newClientID())

	if !h.hub.Join(code, id, conn) {
		conn.Close(websocket.StatusPolicyViolation, "room is full")
		return
	}
	defer h.hub.Leave(code, id)

	ctx := r.Context()
	for {
		_, message, err := conn.Read(ctx)
		if err != nil {
			conn.Close(websocket.StatusNormalClosure, "")
			return
		}
		h.hub.Broadcast(code, id, message)
	}
}
