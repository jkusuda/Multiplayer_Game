CREATE TABLE players (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    display_name VARCHAR(50),
    created_at TIMESTAMPTZ
);

CREATE TABLE rooms (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    host_player_id INT,
    FOREIGN KEY (host_player_id) REFERENCES players(id),
    created_at TIMESTAMPTZ,
    room_status VARCHAR(50)
);

CREATE TABLE room_players (
    room_id INT REFERENCES rooms(id) ON DELETE CASCADE,
    player_id INT REFERENCES players(id),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    PRIMARY KEY (room_id, player_id)
);