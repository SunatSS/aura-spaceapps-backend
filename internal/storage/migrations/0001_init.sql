-- Initial schema. Timestamps are ISO-8601 UTC strings (TEXT), which SQLite
-- compares correctly as strings.

CREATE TABLE missions (
    id          INTEGER PRIMARY KEY,
    name        TEXT NOT NULL,
    -- iss | lunar_surface | mars_transit | mars_surface
    environment TEXT NOT NULL,
    started_at  TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE astronauts (
    id          INTEGER PRIMARY KEY,
    mission_id  INTEGER REFERENCES missions(id) ON DELETE SET NULL,
    name        TEXT NOT NULL,
    -- crew | commander | flight_surgeon
    role        TEXT NOT NULL DEFAULT 'crew',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE measurements (
    id           INTEGER PRIMARY KEY,
    astronaut_id INTEGER NOT NULL REFERENCES astronauts(id) ON DELETE CASCADE,
    -- indicator code from the catalog, e.g. heart_rate, bp_systolic, glucose
    indicator    TEXT NOT NULL,
    value        REAL NOT NULL,
    unit         TEXT NOT NULL,
    measured_at  TEXT NOT NULL,
    -- manual | device | questionnaire | seed
    source       TEXT NOT NULL DEFAULT 'manual',
    note         TEXT,
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_measurements_astronaut_indicator_time
    ON measurements (astronaut_id, indicator, measured_at);
