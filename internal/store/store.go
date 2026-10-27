// Package store is the only code that talks to Postgres.
// This file holds connecting and running migrations.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"time"

	_ "github.com/lib/pq" // Postgres driver for database/sql

	"backend/migrations"
)

type Store struct {
	db *sql.DB
}

// Open connects to Postgres and checks the connection works.
func Open(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	s := &Store{db: db}
	if err := s.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return s, nil
}

// Close closes the database connections.
func (s *Store) Close() error { return s.db.Close() }

// Ping checks the database is reachable.
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// migrationLock is an arbitrary advisory lock id, so two servers starting at
// once cannot run the same migration twice.
const migrationLock = 741100

// Migrate runs every migrations/*.sql file that has not run yet, in name order,
// and records each one in schema_migrations.
func (s *Store) Migrate(ctx context.Context) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, migrationLock); err != nil {
		return err
	}
	defer conn.ExecContext(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, migrationLock)

	if _, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    text PRIMARY KEY,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}

	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, name := range files {
		var done bool
		if err := conn.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, name,
		).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}

		body, err := migrations.FS.ReadFile(name)
		if err != nil {
			return err
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, string(body)); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
