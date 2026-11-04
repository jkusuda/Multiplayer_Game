package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"backend/internal/store"
)

// newTestServer serves the real API on a real database. It skips the test if
// TEST_DATABASE_URL is not set.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	st, err := store.Open(context.Background(), url)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	if err := st.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	srv := httptest.NewServer(New(st, slog.New(slog.NewTextHandler(io.Discard, nil))))
	t.Cleanup(srv.Close)
	return srv
}

// post sends body as JSON and decodes the reply into a map.
func post(t *testing.T, srv *httptest.Server, path string, body any) (int, map[string]any) {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	reply := map[string]any{}
	if resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&reply); err != nil {
			t.Fatalf("POST %s: decode reply: %v", path, err)
		}
	}
	return resp.StatusCode, reply
}

// expect fails the test unless a reply has this status and, for errors, this code.
func expect(t *testing.T, step string, status int, reply map[string]any, wantStatus int, wantError string) {
	t.Helper()
	if status != wantStatus {
		t.Fatalf("%s: status = %d, want %d (reply %v)", step, status, wantStatus, reply)
	}
	if wantError != "" && reply["error"] != wantError {
		t.Fatalf("%s: error = %v, want %s", step, reply["error"], wantError)
	}
}

func createPlayer(t *testing.T, srv *httptest.Server, name string) int64 {
	t.Helper()
	status, reply := post(t, srv, "/players", map[string]any{"display_name": name})
	expect(t, "create player", status, reply, http.StatusCreated, "")
	return int64(reply["id"].(float64))
}

func createRoom(t *testing.T, srv *httptest.Server, hostID int64, maxPlayers int) string {
	t.Helper()
	status, reply := post(t, srv, "/rooms", map[string]any{
		"host_player_id":  hostID,
		"relay_join_code": "RELAY1",
		"max_players":     maxPlayers,
	})
	expect(t, "create room", status, reply, http.StatusCreated, "")
	if reply["relay_join_code"] != "RELAY1" || reply["status"] != "waiting" {
		t.Fatalf("create room: unexpected reply %v", reply)
	}
	return reply["code"].(string)
}

func TestRoomFlow(t *testing.T) {
	srv := newTestServer(t)
	host := createPlayer(t, srv, "host")
	guest := createPlayer(t, srv, "guest")
	code := createRoom(t, srv, host, 4)

	joinPath := "/rooms/" + code + "/join"
	leavePath := "/rooms/" + code + "/leave"

	status, reply := post(t, srv, joinPath, map[string]any{"player_id": guest})
	expect(t, "join", status, reply, http.StatusOK, "")
	if reply["relay_join_code"] != "RELAY1" {
		t.Fatalf("join: relay_join_code = %v", reply["relay_join_code"])
	}

	status, reply = post(t, srv, joinPath, map[string]any{"player_id": guest})
	expect(t, "join twice", status, reply, http.StatusOK, "")

	status, reply = post(t, srv, leavePath, map[string]any{"player_id": guest})
	expect(t, "leave", status, reply, http.StatusNoContent, "")

	status, reply = post(t, srv, leavePath, map[string]any{"player_id": guest})
	expect(t, "leave twice", status, reply, http.StatusNotFound, "player_not_found")

	// Rejoining reuses the player's old room_players row.
	status, reply = post(t, srv, joinPath, map[string]any{"player_id": guest})
	expect(t, "rejoin", status, reply, http.StatusOK, "")

	// The room closes when its host leaves.
	status, reply = post(t, srv, leavePath, map[string]any{"player_id": host})
	expect(t, "host leaves", status, reply, http.StatusNoContent, "")

	status, reply = post(t, srv, joinPath, map[string]any{"player_id": guest})
	expect(t, "join closed room", status, reply, http.StatusConflict, "room_closed")
}

func TestRoomFull(t *testing.T) {
	srv := newTestServer(t)
	host := createPlayer(t, srv, "host")
	code := createRoom(t, srv, host, 4)
	joinPath := "/rooms/" + code + "/join"

	for i := 1; i < 4; i++ {
		player := createPlayer(t, srv, fmt.Sprintf("guest %d", i))
		status, reply := post(t, srv, joinPath, map[string]any{"player_id": player})
		expect(t, "join", status, reply, http.StatusOK, "")
	}

	late := createPlayer(t, srv, "late")
	status, reply := post(t, srv, joinPath, map[string]any{"player_id": late})
	expect(t, "join full room", status, reply, http.StatusConflict, "room_full")

	// A player who is already in the room can still get back in when it is full.
	status, reply = post(t, srv, joinPath, map[string]any{"player_id": host})
	expect(t, "host rejoins full room", status, reply, http.StatusOK, "")
}

func TestRoomErrors(t *testing.T) {
	srv := newTestServer(t)
	host := createPlayer(t, srv, "host")
	const missingPlayer = 2147483647

	tests := []struct {
		name       string
		path       string
		body       map[string]any
		wantStatus int
		wantError  string
	}{
		{"unknown host", "/rooms", map[string]any{"host_player_id": missingPlayer, "relay_join_code": "RELAY1", "max_players": 4}, http.StatusNotFound, "player_not_found"},
		{"bad host id", "/rooms", map[string]any{"host_player_id": 0, "relay_join_code": "RELAY1", "max_players": 4}, http.StatusBadRequest, "invalid_host_player_id"},
		{"too few players", "/rooms", map[string]any{"host_player_id": host, "relay_join_code": "RELAY1", "max_players": 3}, http.StatusBadRequest, "invalid_max_players"},
		{"too many players", "/rooms", map[string]any{"host_player_id": host, "relay_join_code": "RELAY1", "max_players": 9}, http.StatusBadRequest, "invalid_max_players"},
		{"no relay code", "/rooms", map[string]any{"host_player_id": host, "relay_join_code": "", "max_players": 4}, http.StatusBadRequest, "invalid_relay_join_code"},
		{"long relay code", "/rooms", map[string]any{"host_player_id": host, "relay_join_code": "ABCDEFGHIJK", "max_players": 4}, http.StatusBadRequest, "invalid_relay_join_code"},
		{"bad room code", "/rooms/abc/join", map[string]any{"player_id": host}, http.StatusBadRequest, "invalid_code"},
		{"bad room code on leave", "/rooms/abc/leave", map[string]any{"player_id": host}, http.StatusBadRequest, "invalid_code"},
		{"bad player id", "/rooms/ABCDE/join", map[string]any{"player_id": 0}, http.StatusBadRequest, "invalid_player_id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, reply := post(t, srv, tc.path, tc.body)
			expect(t, tc.name, status, reply, tc.wantStatus, tc.wantError)
		})
	}

	t.Run("unknown player joins", func(t *testing.T) {
		code := createRoom(t, srv, host, 4)
		status, reply := post(t, srv, "/rooms/"+code+"/join", map[string]any{"player_id": missingPlayer})
		expect(t, "join", status, reply, http.StatusNotFound, "player_not_found")
	})
}
