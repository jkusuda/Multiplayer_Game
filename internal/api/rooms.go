// Room endpoints: create, join, leave

package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"backend/internal/store"
)

// TODO(Helen): add RelayJoinCode `json:"relay_join_code"` and MaxPlayers
// `json:"max_players"`, check them in createRoom, and pass them to the store.
type createRoomRequest struct {
	HostPlayerID  int64  `json:"host_player_id"`
	RelayJoinCode string `json:"relay_join_code"`
	MaxPlayers    int64  `json:"max_players"`
}

type roomResponse struct {
	ID            int64  `json:"id"`
	Code          string `json:"code"`
	HostPlayerID  int64  `json:"host_player_id"`
	Status        string `json:"status"`
	RelayJoinCode string `json:"relay_join_code`
	MaxPlayers    int64  `json:"max_players`
}

type joinRoomRequest struct {
	Code     string `json:"code"`
	PlayerID int64  `json:"player_id"`
}

type joinRoomResponse struct {
	Code          string `json:"code"`
	RelayJoinCode string `json:"relay_join_code"`
	HostPlayerID  int64  `json:"host_player_id"`
}

type leaveRoomRequest struct {
	PlayerID int64 `json:"player_id"`
}

// createRoom handles POST /rooms: opens a room and replies with its code.
func (a *API) createRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.HostPlayerID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_host_player_id", "host_player_id must be positive")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	room, err := a.store.CreateRoom(ctx, req.HostPlayerID, req.RelayJoinCode, req.MaxPlayers)
	if errors.Is(err, store.ErrHostNotFound) {
		writeError(w, http.StatusNotFound, "player_not_found", "no player has that host_player_id")
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, roomResponse{
		ID:            room.ID,
		Code:          room.Code,
		HostPlayerID:  room.HostPlayerID,
		Status:        room.Status,
		RelayJoinCode: room.RelayJoinCode,
		MaxPlayers:    room.MaxPlayers,
	})
}

// joinRoom handles POST /rooms/{code}/join: adds a player and replies with
// the relay join code Unity needs to connect to the host.
func (a *API) joinRoom(w http.ResponseWriter, r *http.Request) {
	var req joinRoomRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.PlayerID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_host_player_id", "host_player_id must be positive")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	room, err := a.store.JoinRoom(ctx, req.Code, req.PlayerID)
	if errors.Is(err, store.ErrRoomNotFound) {
		writeError(w, http.StatusNotFound, "room_not_found", "no room has that code")
		return
	}
	if errors.Is(err, store.ErrPlayerNotFound) {
		writeError(w, http.StatusNotFound, "player_not_found", "no player has that id")
		return
	}
	if errors.Is(err, store.ErrRoomClosed) {
		writeError(w, http.StatusConflict, "room_closed", "room with that code is closed")
		return
	}
	if errors.Is(err, store.ErrRoomFull) {
		writeError(w, http.StatusConflict, "room_full", "room is at player max")
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, joinRoomResponse{
		Code:          room.Code,
		RelayJoinCode: room.RelayJoinCode,
		HostPlayerID:  room.HostPlayerID,
	})
}

// leaveRoom handles POST /rooms/{code}/leave: marks a player as gone.
//
// TODO - Helen :D
func (a *API) leaveRoom(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "POST /rooms/{code}/leave is not built yet")
}
