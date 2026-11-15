-- Mission phases (environment changes during a mission), activities such as
-- EVA, and measurement context. Roles are now:
-- crew | commander | deputy_commander | flight_surgeon.

CREATE TABLE mission_phases (
    id          INTEGER PRIMARY KEY,
    mission_id  INTEGER NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    -- iss | lunar_surface | mars_transit | mars_surface
    environment TEXT NOT NULL,
    starts_at   TEXT NOT NULL,
    ends_at     TEXT
);

CREATE INDEX idx_mission_phases_mission ON mission_phases (mission_id, starts_at);

CREATE TABLE activities (
    id           INTEGER PRIMARY KEY,
    astronaut_id INTEGER NOT NULL REFERENCES astronauts(id) ON DELETE CASCADE,
    -- eva | landing
    kind         TEXT NOT NULL,
    starts_at    TEXT NOT NULL,
    ends_at      TEXT,
    note         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_activities_astronaut_time ON activities (astronaut_id, starts_at);

-- '' | pre_eva | post_eva
ALTER TABLE measurements ADD COLUMN context TEXT NOT NULL DEFAULT '';
ALTER TABLE measurements ADD COLUMN activity_id INTEGER REFERENCES activities(id) ON DELETE SET NULL;
