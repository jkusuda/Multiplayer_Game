package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math/big"
)

type Room struct {
	ID           int64
	Code         string
	HostPlayerID int64
	Status       string
}

const roomCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ"

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

func (s *Store) CreateRoom(ctx context.Context, hostPlayerID int64) (Room, error) {
	maxCodeAttempts := 10
	var ErrRoomCodeExhausted = errors.New("could not generate a unique room code")
	for attempt := 0; attempt < maxCodeAttempts; attempt++ {
		var room Room
		code, err := generateRoomCode()
		err = s.db.QueryRowContext(ctx, `
		INSERT INTO rooms (code, host_player_id, room_status)
		VALUES ($1, $2, 'waiting')
		RETURNING id, code, host_player_id, room_status
	`, code, hostPlayerID).Scan(
			&room.ID,
			&room.Code,
			&room.HostPlayerID,
			&room.Status,
		)
		if err == nil {
			return room, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		return Room{}, fmt.Errorf("insert room: %w", err)
	}
	log.Printf("Error occured: too many attempts to generate room code. Try again.")
	return Room{}, ErrRoomCodeExhausted
}
