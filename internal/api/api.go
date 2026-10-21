// Package api turns HTTP requests into calls on the store and JSON replies.
//
// This file is plumbing: the route list, one working endpoint, and helpers.
// Your handlers go in new files in this folder, as methods on API. For example:
//
//	func (a *API) createRoom(w http.ResponseWriter, r *http.Request) {
//		var req createRoomRequest
//		if !readJSON(w, r, &req) {
//			return
//		}
//		// check the input, call a.store, then writeJSON or writeError
//	}
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

// New returns the whole HTTP API. Register every route here.
func New(st *store.Store, log *slog.Logger) http.Handler {
	a := &API{store: st, log: log}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.health)

	// TODO: your routes. The pattern is "METHOD /path", and {name} captures part
	// of the path, which a handler reads with r.PathValue("name"). For example:
	//
	//	mux.HandleFunc("POST /rooms", a.createRoom)
	//	mux.HandleFunc("POST /rooms/{code}/join", a.joinRoom)
	mux.HandleFunc("POST /rooms", a.createRoom)

	return a.logRequests(mux)
}

// health is the one finished endpoint. It shows the full path of a request:
// handler -> store -> database -> JSON reply.
func (a *API) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_down", "cannot reach the database")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---- helpers for your handlers ----

// errorBody is the shape of every error reply: a short code for the game to
// check, and a message for a human to read.
type errorBody struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// writeJSON sends body as JSON with the given HTTP status.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// writeError sends an error reply, for example
// writeError(w, http.StatusNotFound, "room_not_found", "no room has that code").
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorBody{Error: code, Message: message})
}

// serverError is for failures that are our fault (a query broke, the database
// is down). It logs the real error and tells the client only that something broke.
func (a *API) serverError(w http.ResponseWriter, r *http.Request, err error) {
	a.log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "something went wrong on the server")
}

// readJSON decodes the request body into dst. On bad input it replies 400 and
// returns false, so the handler should just return.
func readJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10) // refuse bodies over 64 KB
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json", "request body is not valid JSON")
		return false
	}
	return true
}

// statusRecorder remembers the status code so the log line can include it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// logRequests prints one line per request: method, path, status and time taken.
func (a *API) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		a.log.Info("request", "method", r.Method, "path", r.URL.Path,
			"status", rec.status, "ms", time.Since(start).Milliseconds())
	})
}
