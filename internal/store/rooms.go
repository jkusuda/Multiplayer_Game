// Room queries: create, join, leave.

package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/lib/pq"
	"github.com/lib/pq/pqerror"
)

type Room struct {
	ID           int64
	Code         string
	HostPlayerID int64
	Status       string // "waiting" or "closed"

	// TODO - Helen: add migrations/0002_room_relay_code.sql with
	//   ALTER TABLE rooms ADD COLUMN relay_join_code VARCHAR(10);
	//   ALTER TABLE rooms ADD COLUMN max_players INT NOT NULL DEFAULT 4;
	// then read and write these two fields in the queries below.
	RelayJoinCode string
	MaxPlayers    int
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
//
// TODO - Helen :D
// take relayJoinCode and maxPlayers too, save them in the INSERT,
// and add the host to room_players in the same transaction.
func (s *Store) CreateRoom(ctx context.Context, hostPlayerID int64) (Room, error) {
	for attempt := 0; attempt < maxRoomCodeAttempts; attempt++ {
		code, err := generateRoomCode()
		if err != nil {
			return Room{}, fmt.Errorf("generate room code: %w", err)
		}

		var room Room
		err = s.db.QueryRowContext(ctx, `
			INSERT INTO rooms (code, host_player_id, room_status, created_at)
			VALUES ($1, $2, 'waiting', now())
			RETURNING id, code, host_player_id, room_status
		`, code, hostPlayerID).Scan(&room.ID, &room.Code, &room.HostPlayerID, &room.Status)
		if err == nil {
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
//
// TODO - Helen :D
func (s *Store) JoinRoom(ctx context.Context, code string, playerID int64) (Room, error) {
	return Room{}, errNotImplemented
}

// LeaveRoom marks playerID as gone from the room with this code. If the host
// leaves, the room closes, because its Relay game ends with the host.
//
// TODO - Helen :D
func (s *Store) LeaveRoom(ctx context.Context, code string, playerID int64) error {
	return errNotImplemented
}
