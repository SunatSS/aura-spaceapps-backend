-- Crew member particularities (REQUIREMENTS FR-6, FR-24) and daily task
-- completions (FR-41).

ALTER TABLE astronauts ADD COLUMN conditions TEXT NOT NULL DEFAULT '';

-- Preflight baseline and personal ranges per indicator.
CREATE TABLE personal_indicators (
    astronaut_id INTEGER NOT NULL REFERENCES astronauts(id) ON DELETE CASCADE,
    indicator    TEXT NOT NULL,
    baseline     REAL,
    -- personal ranges, used when has_ranges = 1
    nominal_min  REAL,
    nominal_max  REAL,
    caution_min  REAL,
    caution_max  REAL,
    has_ranges   INTEGER NOT NULL DEFAULT 0,
    note         TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (astronaut_id, indicator)
);

-- Personal system weights in the Readiness Score.
CREATE TABLE personal_weights (
    astronaut_id INTEGER NOT NULL REFERENCES astronauts(id) ON DELETE CASCADE,
    system       TEXT NOT NULL,
    weight       REAL NOT NULL,
    PRIMARY KEY (astronaut_id, system)
);

-- Manual tasks marked done, one row per task and mission-UTC day.
CREATE TABLE task_completions (
    astronaut_id INTEGER NOT NULL REFERENCES astronauts(id) ON DELETE CASCADE,
    task         TEXT NOT NULL,
    day          TEXT NOT NULL,
    completed_at TEXT NOT NULL,
    PRIMARY KEY (astronaut_id, task, day)
);
