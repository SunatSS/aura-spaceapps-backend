-- Passkeys (REQUIREMENTS FR-74): WebAuthn credentials and single-use challenges.

CREATE TABLE passkeys (
    -- credential ID, base64url
    id           TEXT PRIMARY KEY,
    astronaut_id INTEGER NOT NULL REFERENCES astronauts(id) ON DELETE CASCADE,
    -- COSE_Key as sent by the authenticator
    public_key   BLOB NOT NULL,
    alg          INTEGER NOT NULL,
    sign_count   INTEGER NOT NULL DEFAULT 0,
    label        TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    last_used_at TEXT
);

CREATE INDEX idx_passkeys_astronaut ON passkeys (astronaut_id);

CREATE TABLE webauthn_challenges (
    challenge    TEXT PRIMARY KEY,
    -- set for registration (the signed-in member); NULL for sign-in
    astronaut_id INTEGER REFERENCES astronauts(id) ON DELETE CASCADE,
    -- register | login
    kind         TEXT NOT NULL,
    expires_at   TEXT NOT NULL
);
