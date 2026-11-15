-- Vehicle systems (REQUIREMENTS §5.9) and medical supplies (§5.10).

-- Planned mission end, for supply forecasts (FR-94).
ALTER TABLE missions ADD COLUMN planned_end_at TEXT;

-- Habitat modules and where each member is now (FR-83).
CREATE TABLE modules (
    id         INTEGER PRIMARY KEY,
    mission_id INTEGER NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    code       TEXT NOT NULL,
    name       TEXT NOT NULL,
    sort       INTEGER NOT NULL DEFAULT 0,
    UNIQUE (mission_id, code)
);

ALTER TABLE astronauts ADD COLUMN module_code TEXT NOT NULL DEFAULT '';

-- Environment telemetry from onboard sensors (FR-16, FR-82).
CREATE TABLE telemetry (
    id          INTEGER PRIMARY KEY,
    mission_id  INTEGER NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    module_code TEXT NOT NULL,
    -- vehicle parameter code from the catalog, e.g. ppco2
    parameter   TEXT NOT NULL,
    value       REAL NOT NULL,
    unit        TEXT NOT NULL,
    measured_at TEXT NOT NULL,
    source      TEXT NOT NULL DEFAULT 'device'
);

CREATE INDEX idx_telemetry_module_parameter_time ON telemetry (mission_id, module_code, parameter, measured_at);

-- Medical supplies (FR-90): one lot per item.
CREATE TABLE supplies (
    id              INTEGER PRIMARY KEY,
    mission_id      INTEGER NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    code            TEXT NOT NULL,
    name            TEXT NOT NULL,
    -- medication | consumable | equipment | oxygen
    category        TEXT NOT NULL,
    unit            TEXT NOT NULL,
    quantity        REAL NOT NULL,
    min_quantity    REAL NOT NULL DEFAULT 0,
    location        TEXT NOT NULL,
    expires_at      TEXT,
    last_counted_at TEXT,
    -- counted minus expected at the last inventory count (FR-95)
    discrepancy     REAL NOT NULL DEFAULT 0,
    UNIQUE (mission_id, code)
);

-- Supply journal (FR-92): who took how much, when and why.
CREATE TABLE supply_transactions (
    id                INTEGER PRIMARY KEY,
    supply_id         INTEGER NOT NULL REFERENCES supplies(id) ON DELETE CASCADE,
    astronaut_id      INTEGER REFERENCES astronauts(id) ON DELETE SET NULL,
    -- use | count | dispose
    kind              TEXT NOT NULL,
    delta             REAL NOT NULL,
    quantity_after    REAL NOT NULL,
    reason            TEXT NOT NULL DEFAULT '',
    recommendation_id TEXT NOT NULL DEFAULT '',
    created_at        TEXT NOT NULL
);

CREATE INDEX idx_supply_transactions_supply_time ON supply_transactions (supply_id, created_at);
