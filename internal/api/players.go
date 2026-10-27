// Player endpoints: create and look up.

package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"backend/internal/store"
)

const maxDisplayNameLength = 50 // matches players.display_name VARCHAR(50)

type createPlayerRequest struct {
	DisplayName string `json:"display_name"`
}

type playerResponse struct {
	ID          int64     `json:"id"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// createPlayer handles POST /players: adds a player and replies with its id.
func (a *API) createPlayer(w http.ResponseWriter, r *http.Request) {
	var req createPlayerRequest
	if !readJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.DisplayName)
	if name == "" || utf8.RuneCountInString(name) > maxDisplayNameLength {
		writeError(w, http.StatusBadRequest, "invalid_display_name", "display_name must be 1 to 50 characters")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	player, err := a.store.CreatePlayer(ctx, name)
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, toPlayerResponse(player))
}

// getPlayer handles GET /players/{id}: replies with one player.
func (a *API) getPlayer(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_player_id", "player id must be a positive number")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	player, err := a.store.GetPlayer(ctx, id)
	if errors.Is(err, store.ErrPlayerNotFound) {
		writeError(w, http.StatusNotFound, "player_not_found", "no player has that id")
		return
	}
	if err != nil {
		a.serverError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toPlayerResponse(player))
}

// toPlayerResponse converts a stored player into the JSON reply.
func toPlayerResponse(p store.Player) playerResponse {
	return playerResponse{ID: p.ID, DisplayName: p.DisplayName, CreatedAt: p.CreatedAt}
}
