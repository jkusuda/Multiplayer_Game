// Room queries: create, join, leave.

package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

type Room struct {
	ID            int64
	Code          string
	HostPlayerID  int64
	Status        string // "waiting" or "closed"
	RelayJoinCode string
	MaxPlayers    int64
}

var (
	ErrHostNotFound      = errors.New("host player does not exist")
	ErrRoomCodeExhausted = errors.New("could not generate a unique room code")
	ErrRoomNotFound      = errors.New("no room has that code")
	ErrRoomClosed        = errors.New("room is closed")
	ErrRoomFull          = errors.New("room is full")
	errNotImplemented    = errors.New("not implemented")
)

// No I or O, so codes cannot be misread as 1 or 0.
const roomCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ"

const maxRoomCodeAttempts = 10

// generateRoomCode returns a random 5-letter code.
func generateRoomCode() (string, error) {
	code := make([]byte, 5)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(roomCodeAlphabet))))
		if err != nil {
			return "", err
		}
		code[i] = roomCodeAlphabet[n.Int64()]
	}
	return string(code), nil
}

// CreateRoom opens a room under a new code, retrying if the code is taken.
// It returns ErrHostNotFound if hostPlayerID is not a player.
func (s *Store) CreateRoom(ctx context.Context, hostPlayerID int64, relayJoinCode string, maxPlayers int64) (Room, error) {
	for attempt := 0; attempt < maxRoomCodeAttempts; attempt++ {
		code, err := generateRoomCode()
		if err != nil {
			return Room{}, fmt.Errorf("generate room code: %w", err)
		}

		var room Room
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO rooms (code, host_player_id, created_at, room_status, relay_join_code, max_players)
			VALUES ($1, $2, now(), 'waiting', $3, $4)
			RETURNING id, code, host_player_id, created_at, room_status, relay_join_code, max_players
		`, code, hostPlayerID, relayJoinCode, maxPlayers).Scan(
			&room.ID,
			&room.Code,
			&room.HostPlayerID,
			&room.Status,
			&room.RelayJoinCode,
			&room.MaxPlayers)
		if err == nil {
			_, err = s.db.ExecContext(ctx, `
					INSERT INTO room_players (room_id, player_id, joined_at, left_at)
					VALUES ($1, $2, now(), NULL)
				`, room.ID, hostPlayerID)
			return room, nil
		}

		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch {
			case pqErr.Code == pqerror.UniqueViolation && pqErr.Constraint == "rooms_code_key":
				continue
			case pqErr.Code == pqerror.ForeignKeyViolation:
				return Room{}, ErrHostNotFound
			}
		}
		return Room{}, fmt.Errorf("insert room: %w", err)
	}
	return Room{}, ErrRoomCodeExhausted
}

// JoinRoom adds playerID to the room with this code and returns the room,
// so the caller gets its relay join code.
func (s *Store) JoinRoom(ctx context.Context, code string, playerID int64) (Room, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Room{}, fmt.Errorf("begin join transaction: %w", err)
	}
	defer tx.Rollback()

	var room Room
	err = tx.QueryRowContext(ctx, `
			SELECT id, code, host_player_id, room_status, relay_join_code, max_players
			FROM rooms
			WHERE code = $1
			FOR UPDATE
		`, code).Scan(
		&room.ID,
		&room.Code,
		&room.HostPlayerID,
		&room.Status,
		&room.RelayJoinCode,
		&room.MaxPlayers)
	if errors.Is(err, sql.ErrNoRows) {
		return Room{}, ErrRoomNotFound
	}
	if err != nil {
		return Room{}, fmt.Errorf("find room: %w", err)
	}
	if room.Status == "closed" {
		return Room{}, ErrRoomClosed
	}

	var alreadyJoined bool
	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM room_players
			WHERE room_id = $1
			AND player_id = $2
			AND left_at IS NULL
		)
	`, room.ID, playerID).Scan(&alreadyJoined)
	if err != nil {
		return Room{}, fmt.Errorf("check room membership: %w", err)
	}

	if alreadyJoined {
		if err := tx.Commit(); err != nil {
			return Room{}, fmt.Errorf("commit existing membership: %w", err)
		}
		return room, nil
	}

	var n int64
	err = tx.QueryRowContext(ctx, `
		SELECT count(*)
		FROM room_players
		WHERE room_id = $1 
			AND left_at IS NULL
		`, room.ID).Scan(&n)
	if err != nil {
		return Room{}, fmt.Errorf("count room players: %w", err)
	}
	if n >= room.MaxPlayers {
		return Room{}, ErrRoomFull
	}

	_, err = tx.ExecContext(ctx, `
			INSERT INTO room_players (room_id, player_id, joined_at, left_at)
			VALUES ($1, $2, now(), NULL)
		`, room.ID, playerID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) &&
			pqErr.Code == pqerror.ForeignKeyViolation &&
			pqErr.Constraint == "room_players_player_id_fkey" {
			return Room{}, ErrPlayerNotFound
		}
		return Room{}, fmt.Errorf("insert player: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Room{}, fmt.Errorf("commit join transaction: %w", err)
	}

	return room, nil
}

// LeaveRoom marks playerID as gone from the room with this code. If the host
// leaves, the room closes, because its Relay game ends with the host.
//
// TODO - Helen :D
func (s *Store) LeaveRoom(ctx context.Context, code string, playerID int64) error {
	return errNotImplemented
}
