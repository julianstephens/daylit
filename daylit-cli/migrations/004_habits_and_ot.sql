-- Migration 004: Add Habits and Once-Today (OT) tracking
-- Adds support for habit tracking and daily OT (Once-Today) intentions

-- Table: habits
-- Stores habit definitions
CREATE TABLE IF NOT EXISTS habits (
    id          TEXT PRIMARY KEY,        -- UUID
    name        TEXT NOT NULL UNIQUE,
    created_at  TEXT NOT NULL,           -- ISO8601
    archived_at TEXT NULL,               -- archived but not deleted
    deleted_at  TEXT NULL                -- soft-delete marker
);

-- Table: habit_entries
-- Records when habits were performed
CREATE TABLE IF NOT EXISTS habit_entries (
    id         TEXT PRIMARY KEY,
    habit_id   TEXT NOT NULL REFERENCES habits(id),
    day        TEXT NOT NULL,           -- YYYY-MM-DD (aligned to daylit's timezone)
    note       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT NULL,
    UNIQUE(habit_id, day)
);

-- Table: ot_settings
-- Configuration for OT feature (single row table)
CREATE TABLE IF NOT EXISTS ot_settings (
    id               INTEGER PRIMARY KEY CHECK (id = 1),
    prompt_on_empty  INTEGER NOT NULL,   -- boolean 0/1
    strict_mode      INTEGER NOT NULL,   -- boolean 0/1
    default_log_days INTEGER NOT NULL
);

-- Table: ot_entries
-- Stores daily OT (Once-Today) intentions
CREATE TABLE IF NOT EXISTS ot_entries (
    id         TEXT PRIMARY KEY,
    day        TEXT NOT NULL UNIQUE,
    title      TEXT NOT NULL,
    note       TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT NULL
);

-- Seed ot_settings with default values if not present
INSERT OR IGNORE INTO ot_settings (id, prompt_on_empty, strict_mode, default_log_days)
VALUES (1, 1, 1, 14);
