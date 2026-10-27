// Command server starts the game backend. It reads PORT (default 8080) and
// DATABASE_URL (default matches docker-compose.yml) from the environment.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/api"
	"backend/internal/store"
)

// main connects to Postgres, runs migrations, and serves HTTP until Ctrl+C.
func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	port := envOr("PORT", "8080")
	dbURL := envOr("DATABASE_URL", "postgres://game:game@localhost:5432/game?sslmode=disable")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, dbURL)
	if err != nil {
		log.Error("cannot open database", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	if err := st.Migrate(ctx); err != nil {
		log.Error("cannot run migrations", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           api.New(st, log),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	go func() {
		log.Info("listening", "port", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
	log.Info("stopped")
}

// envOr returns the environment variable key, or fallback if it is unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
