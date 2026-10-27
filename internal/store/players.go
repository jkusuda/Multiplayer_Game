// Player queries: create and look up.

package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Player struct {
	ID          int64
	DisplayName string
	CreatedAt   time.Time
}

var ErrPlayerNotFound = errors.New("player does not exist")

// CreatePlayer adds a player and returns it with its new id.
func (s *Store) CreatePlayer(ctx context.Context, displayName string) (Player, error) {
	var p Player
	err := s.db.QueryRowContext(ctx, `
		INSERT INTO players (display_name, created_at)
		VALUES ($1, now())
		RETURNING id, display_name, created_at
	`, displayName).Scan(&p.ID, &p.DisplayName, &p.CreatedAt)
	if err != nil {
		return Player{}, fmt.Errorf("insert player: %w", err)
	}
	return p, nil
}

// GetPlayer returns one player, or ErrPlayerNotFound.
func (s *Store) GetPlayer(ctx context.Context, id int64) (Player, error) {
	var p Player
	err := s.db.QueryRowContext(ctx, `
		SELECT id, display_name, created_at
		FROM players
		WHERE id = $1
	`, id).Scan(&p.ID, &p.DisplayName, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Player{}, ErrPlayerNotFound
	}
	if err != nil {
		return Player{}, fmt.Errorf("select player: %w", err)
	}
	return p, nil
}
