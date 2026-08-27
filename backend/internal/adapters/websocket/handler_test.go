package websocket

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"horda-zumbi/backend/internal/adapters/httpserver"
	"horda-zumbi/backend/internal/adapters/memory"
)

const testTimeout = 2 * time.Second

// newTestServer starts a real HTTP server exposing the websocket room
// endpoint, backed by a fresh Hub and in-memory RoomRepository.
func newTestServer(t *testing.T) (*httptest.Server, *memory.RoomRepository) {
	t.Helper()

	roomRepo := memory.NewRoomRepository()
	hub := NewHub(roomRepo)
	handler := NewHandler(hub)
	server := httptest.NewServer(httpserver.NewServer(handler))
	t.Cleanup(server.Close)

	return server, roomRepo
}

// dialRoom opens a real websocket connection to the given room code on
// server.
func dialRoom(t *testing.T, server *httptest.Server, code string) *websocket.Conn {
	t.Helper()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/rooms/" + code
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	t.Cleanup(func() { conn.Close(websocket.StatusNormalClosure, "") })

	return conn
}

// readMessage reads a single message from conn, failing the test if none
// arrives within testTimeout.
func readMessage(t *testing.T, conn *websocket.Conn) []byte {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	_, msg, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	return msg
}

// assertNoMessage confirms no message arrives on conn within a short
// window. The read timeout closes conn as a side effect, so it must be the
// last use of conn in a test.
func assertNoMessage(t *testing.T, conn *websocket.Conn) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	_, msg, err := conn.Read(ctx)
	if err == nil {
		t.Errorf("expected no message, got %q", msg)
	}
}

func TestHandler_BroadcastsToOtherClientButNotSender(t *testing.T) {
	server, _ := newTestServer(t)

	alice := dialRoom(t, server, "ROOM1")
	bob := dialRoom(t, server, "ROOM1")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := alice.Write(ctx, websocket.MessageText, []byte("hello from alice")); err != nil {
		t.Fatalf("write: %v", err)
	}

	if got := readMessage(t, bob); string(got) != "hello from alice" {
		t.Errorf("bob received %q, want %q", got, "hello from alice")
	}

	assertNoMessage(t, alice)
}

func TestHandler_BroadcastsToAllOtherClientsInRoom(t *testing.T) {
	server, _ := newTestServer(t)

	alice := dialRoom(t, server, "ROOM2")
	bob := dialRoom(t, server, "ROOM2")
	carol := dialRoom(t, server, "ROOM2")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := alice.Write(ctx, websocket.MessageText, []byte("hi all")); err != nil {
		t.Fatalf("write: %v", err)
	}

	if got := readMessage(t, bob); string(got) != "hi all" {
		t.Errorf("bob received %q, want %q", got, "hi all")
	}
	if got := readMessage(t, carol); string(got) != "hi all" {
		t.Errorf("carol received %q, want %q", got, "hi all")
	}
}

func TestHandler_RejectsFourthClientWhenRoomFull(t *testing.T) {
	server, _ := newTestServer(t)

	dialRoom(t, server, "ROOM3")
	dialRoom(t, server, "ROOM3")
	dialRoom(t, server, "ROOM3")

	fourth := dialRoom(t, server, "ROOM3")

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	_, _, err := fourth.Read(ctx)
	if err == nil {
		t.Fatal("expected fourth client's connection to be closed, got nil error")
	}
	if status := websocket.CloseStatus(err); status != websocket.StatusPolicyViolation {
		t.Errorf("close status = %v, want %v", status, websocket.StatusPolicyViolation)
	}
}
