package websocket

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"horda-zumbi/backend/internal/adapters/httpserver"
	"horda-zumbi/backend/internal/adapters/memory"
	"horda-zumbi/backend/internal/domain"
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

// performJoin writes a join request on conn without waiting for a response.
func performJoin(t *testing.T, conn *websocket.Conn, req joinMessage) {
	t.Helper()

	payload, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal join message: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, payload); err != nil {
		t.Fatalf("write join message: %v", err)
	}
}

// joinRoom performs the JOIN handshake on conn and returns the server's
// "joined" confirmation, failing the test if the join is rejected.
func joinRoom(t *testing.T, conn *websocket.Conn, name string, character domain.CharacterID) joinedMessage {
	t.Helper()

	performJoin(t, conn, joinMessage{Type: "join", Name: name, Character: character})

	raw := readMessage(t, conn)
	var resp joinedMessage
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal joined message: %v (raw=%s)", err, raw)
	}
	if resp.Type != "joined" {
		t.Fatalf("expected joined message, got %q (raw=%s)", resp.Type, raw)
	}
	return resp
}

// expectPlayerJoined reads a "player_joined" event from conn, optionally
// asserting it refers to wantPlayerID (ignored when empty).
func expectPlayerJoined(t *testing.T, conn *websocket.Conn, wantPlayerID string) playerJoinedMessage {
	t.Helper()

	raw := readMessage(t, conn)
	var msg playerJoinedMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal player_joined message: %v (raw=%s)", err, raw)
	}
	if msg.Type != "player_joined" {
		t.Fatalf("expected player_joined message, got %q (raw=%s)", msg.Type, raw)
	}
	if wantPlayerID != "" && msg.Player.ID != wantPlayerID {
		t.Errorf("player_joined for %q, want %q", msg.Player.ID, wantPlayerID)
	}
	return msg
}

// expectJoinRejected reads the "error" message a rejected join produces and
// confirms the connection is then closed with StatusPolicyViolation.
func expectJoinRejected(t *testing.T, conn *websocket.Conn) errorMessage {
	t.Helper()

	raw := readMessage(t, conn)
	var msg errorMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal error message: %v (raw=%s)", err, raw)
	}
	if msg.Type != "error" {
		t.Fatalf("expected error message, got %q (raw=%s)", msg.Type, raw)
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if _, _, err := conn.Read(ctx); err == nil {
		t.Fatal("expected connection to be closed after error message")
	} else if status := websocket.CloseStatus(err); status != websocket.StatusPolicyViolation {
		t.Errorf("close status = %v, want %v", status, websocket.StatusPolicyViolation)
	}

	return msg
}

func TestHandler_JoinReturnsPlayerIDAndInitialRoomSnapshot(t *testing.T) {
	server, _ := newTestServer(t)

	alice := dialRoom(t, server, "ROOM9")
	resp := joinRoom(t, alice, "Alice", domain.CharacterTank)

	if resp.PlayerID == "" {
		t.Error("expected non-empty player_id")
	}
	if resp.Room.Code != "ROOM9" {
		t.Errorf("room code = %q, want %q", resp.Room.Code, "ROOM9")
	}
	if resp.Room.Wave != 0 {
		t.Errorf("wave = %d, want 0", resp.Room.Wave)
	}
	if resp.Room.State != "waiting" {
		t.Errorf("state = %q, want %q", resp.Room.State, "waiting")
	}
	if len(resp.Room.Players) != 1 {
		t.Fatalf("players = %d, want 1", len(resp.Room.Players))
	}
	p := resp.Room.Players[0]
	if p.ID != resp.PlayerID || p.Name != "Alice" || p.Character != domain.CharacterTank {
		t.Errorf("unexpected player in snapshot: %+v", p)
	}
}

func TestHandler_SecondJoinIncludesBothPlayersAndNotifiesFirst(t *testing.T) {
	server, _ := newTestServer(t)

	alice := dialRoom(t, server, "ROOM10")
	joinRoom(t, alice, "Alice", domain.CharacterTank)

	bob := dialRoom(t, server, "ROOM10")
	bobJoined := joinRoom(t, bob, "Bob", domain.CharacterMedic)

	if len(bobJoined.Room.Players) != 2 {
		t.Fatalf("bob's snapshot players = %d, want 2", len(bobJoined.Room.Players))
	}

	notified := expectPlayerJoined(t, alice, bobJoined.PlayerID)
	if notified.Player.Name != "Bob" || notified.Player.Character != domain.CharacterMedic {
		t.Errorf("unexpected player_joined payload: %+v", notified.Player)
	}
}

func TestHandler_RejectsJoinWithEmptyName(t *testing.T) {
	server, _ := newTestServer(t)

	bad := dialRoom(t, server, "ROOM6")
	performJoin(t, bad, joinMessage{Type: "join", Name: "", Character: domain.CharacterTank})
	expectJoinRejected(t, bad)

	// No ghost player was left behind: three valid clients still fit.
	alice := dialRoom(t, server, "ROOM6")
	joinRoom(t, alice, "Alice", domain.CharacterTank)
	bob := dialRoom(t, server, "ROOM6")
	joinRoom(t, bob, "Bob", domain.CharacterMedic)
	expectPlayerJoined(t, alice, "")
	carol := dialRoom(t, server, "ROOM6")
	joinRoom(t, carol, "Carol", domain.CharacterScout)
	expectPlayerJoined(t, alice, "")
	expectPlayerJoined(t, bob, "")
}

func TestHandler_RejectsJoinWithInvalidCharacter(t *testing.T) {
	server, _ := newTestServer(t)

	bad := dialRoom(t, server, "ROOM7")
	performJoin(t, bad, joinMessage{Type: "join", Name: "Mallory", Character: "invalido"})
	expectJoinRejected(t, bad)

	// No ghost player was left behind: three valid clients still fit.
	alice := dialRoom(t, server, "ROOM7")
	joinRoom(t, alice, "Alice", domain.CharacterTank)
	bob := dialRoom(t, server, "ROOM7")
	joinRoom(t, bob, "Bob", domain.CharacterMedic)
	expectPlayerJoined(t, alice, "")
	carol := dialRoom(t, server, "ROOM7")
	joinRoom(t, carol, "Carol", domain.CharacterScout)
	expectPlayerJoined(t, alice, "")
	expectPlayerJoined(t, bob, "")
}

func TestHandler_BroadcastsToOtherClientButNotSender(t *testing.T) {
	server, _ := newTestServer(t)

	alice := dialRoom(t, server, "ROOM1")
	joinRoom(t, alice, "Alice", domain.CharacterTank)
	bob := dialRoom(t, server, "ROOM1")
	joinRoom(t, bob, "Bob", domain.CharacterMedic)
	expectPlayerJoined(t, alice, "")

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
	joinRoom(t, alice, "Alice", domain.CharacterTank)
	bob := dialRoom(t, server, "ROOM2")
	joinRoom(t, bob, "Bob", domain.CharacterMedic)
	expectPlayerJoined(t, alice, "")
	carol := dialRoom(t, server, "ROOM2")
	joinRoom(t, carol, "Carol", domain.CharacterScout)
	expectPlayerJoined(t, alice, "")
	expectPlayerJoined(t, bob, "")

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

	alice := dialRoom(t, server, "ROOM3")
	joinRoom(t, alice, "Alice", domain.CharacterTank)
	bob := dialRoom(t, server, "ROOM3")
	joinRoom(t, bob, "Bob", domain.CharacterMedic)
	expectPlayerJoined(t, alice, "")
	carol := dialRoom(t, server, "ROOM3")
	joinRoom(t, carol, "Carol", domain.CharacterScout)
	expectPlayerJoined(t, alice, "")
	expectPlayerJoined(t, bob, "")

	fourth := dialRoom(t, server, "ROOM3")
	performJoin(t, fourth, joinMessage{Type: "join", Name: "Dave", Character: domain.CharacterTank})
	expectJoinRejected(t, fourth)
}

func TestHandler_DoesNotLeakMessagesBetweenRooms(t *testing.T) {
	server, _ := newTestServer(t)

	aliceInAAAA := dialRoom(t, server, "AAAA")
	joinRoom(t, aliceInAAAA, "Alice", domain.CharacterTank)
	carolInAAAA := dialRoom(t, server, "AAAA")
	joinRoom(t, carolInAAAA, "Carol", domain.CharacterScout)
	expectPlayerJoined(t, aliceInAAAA, "")
	bobInBBBB := dialRoom(t, server, "BBBB")
	joinRoom(t, bobInBBBB, "Bob", domain.CharacterMedic)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := aliceInAAAA.Write(ctx, websocket.MessageText, []byte("only for AAAA")); err != nil {
		t.Fatalf("write: %v", err)
	}

	if got := readMessage(t, carolInAAAA); string(got) != "only for AAAA" {
		t.Errorf("carol (same room) received %q, want %q", got, "only for AAAA")
	}
	assertNoMessage(t, bobInBBBB)
}

// waitForCondition polls condition until it is true, failing the test if it
// is never met within timeout.
func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for {
		if condition() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("condition not met before timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestHandler_RemovesRoomFromRepositoryWhenAllClientsLeave(t *testing.T) {
	server, roomRepo := newTestServer(t)

	conn := dialRoom(t, server, "ROOM5")
	joinRoom(t, conn, "Alice", domain.CharacterTank)

	waitForCondition(t, testTimeout, func() bool {
		_, ok := roomRepo.Get("ROOM5")
		return ok
	})

	conn.Close(websocket.StatusNormalClosure, "")

	waitForCondition(t, testTimeout, func() bool {
		_, ok := roomRepo.Get("ROOM5")
		return !ok
	})
}
