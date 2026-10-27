// Package api turns HTTP requests into store calls and JSON replies.
// This file holds the route list and shared helpers.
package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"backend/internal/store"
)

type API struct {
	store *store.Store
	log   *slog.Logger
}

// New returns the HTTP handler with every route registered.
func New(st *store.Store, log *slog.Logger) http.Handler {
	a := &API{store: st, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)

	mux.HandleFunc("POST /players", a.createPlayer)
	mux.HandleFunc("GET /players/{id}", a.getPlayer)

	mux.HandleFunc("POST /rooms", a.createRoom)
	mux.HandleFunc("POST /rooms/{code}/join", a.joinRoom)
	mux.HandleFunc("POST /rooms/{code}/leave", a.leaveRoom)

	return a.logRequests(mux)
}

// health handles GET /healthz: replies ok if the database answers.
func (a *API) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_down", "cannot reach the database")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// errorBody is the shape of every error reply.
type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// writeJSON sends body as JSON with the given status.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// writeError sends an error reply with a code for the game and a message for people.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: code, Message: message})
}

// serverError logs err and replies 500 without exposing the details.
func (a *API) serverError(w http.ResponseWriter, r *http.Request, err error) {
	a.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "something went wrong on the server")
}

// readJSON decodes the body into dst, or replies 400 and returns false.
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "request body is not valid JSON")
		return false
	}
	return true
}

// statusRecorder remembers the reply status for logRequests.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader records the status, then sends it.
func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// logRequests logs one line per request: method, path, status and time taken.
func (a *API) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		a.log.Info("request", "method", r.Method, "path", r.URL.Path,
			"status", rec.status, "ms", time.Since(start).Milliseconds())
	})
}
