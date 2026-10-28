// Room endpoints: create, join, leave

package api

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"time"

	"backend/internal/store"
)

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
	RelayJoinCode string `json:"relay_join_code"`
	MaxPlayers    int64  `json:"max_players"`
}

type joinRoomRequest struct {
	PlayerID int64 `json:"player_id"`
}

type joinRoomResponse struct {
	Code          string `json:"code"`
	RelayJoinCode string `json:"relay_join_code"`
	HostPlayerID  int64  `json:"host_player_id"`
}

type leaveRoomRequest struct {
	PlayerID int64 `json:"player_id"`
}

var roomCodePattern = regexp.MustCompile(`^[A-HJ-NP-Z]{5}$`)

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
	if req.MaxPlayers < 4 || req.MaxPlayers > 8 {
		writeError(w, http.StatusBadRequest, "invalid_max_players", "max_players must be between 4-8")
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
	code := r.PathValue("code")
	var req joinRoomRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.PlayerID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_player_id", "player_id must be positive")
		return
	}
	if !roomCodePattern.MatchString(code) {
		writeError(w, http.StatusBadRequest, "invalid_code", "room code must be 5 capital letters")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	room, err := a.store.JoinRoom(ctx, code, req.PlayerID)
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
