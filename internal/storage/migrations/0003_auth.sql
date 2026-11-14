-- Sign-in (REQUIREMENTS FR-71): credentials on the mission member and
-- server-side sessions. Only the SHA-256 of a session token is stored.

ALTER TABLE astronauts ADD COLUMN username TEXT;
ALTER TABLE astronauts ADD COLUMN password_hash TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX idx_astronauts_username ON astronauts (username) WHERE username IS NOT NULL;

CREATE TABLE sessions (
    token_hash   TEXT PRIMARY KEY,
    astronaut_id INTEGER NOT NULL REFERENCES astronauts(id) ON DELETE CASCADE,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expires_at   TEXT NOT NULL
);

CREATE INDEX idx_sessions_astronaut ON sessions (astronaut_id);
