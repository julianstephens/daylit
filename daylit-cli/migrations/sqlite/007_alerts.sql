-- Migration 007: Add alerts table
-- Adds support for arbitrary scheduled notifications

CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    message TEXT NOT NULL,
    time TEXT NOT NULL,
    date TEXT,
    recurrence_type TEXT,
    recurrence_interval INTEGER,
    recurrence_weekdays TEXT,
    active BOOLEAN DEFAULT 1,
    last_sent TEXT,
    created_at TEXT NOT NULL
);
