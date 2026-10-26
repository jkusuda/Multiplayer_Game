package api

import (
	"context"
	"net/http"
	"time"
)

type createRoomRequest struct {
	HostPlayerID int64 `json:"host_player_id"`
}

type roomResponse struct {
	ID           int64  `json:"id"`
	Code         string `json:"code"`
	HostPlayerID int64  `json:"host_player_id"`
	Status       string `json:"status"`
}

func (a *API) createRoom(w http.ResponseWriter, r *http.Request) {
	var req createRoomRequest
	if !readJSON(w, r, &req) {
		return
	}
	// check the input, call a.store, then writeJSON or writeError
	if req.HostPlayerID <= 0 {
		writeError(
			w,
			http.StatusBadRequest,
			"invalid_host_player_id",
			"host_player_id must be positive",
		)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	room, err := a.store.CreateRoom(ctx, req.HostPlayerID)
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, roomResponse{
		ID:           room.ID,
		Code:         room.Code,
		HostPlayerID: room.HostPlayerID,
		Status:       room.Status,
	})
}
