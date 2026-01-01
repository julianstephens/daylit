-- Migration 001: Initial schema
-- This migration creates the initial database schema for daylit

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    name TEXT,
    kind TEXT,
    duration_min INTEGER,
    earliest_start TEXT,
    latest_end TEXT,
    fixed_start TEXT,
    fixed_end TEXT,
    recurrence_type TEXT,
    recurrence_interval INTEGER,
    recurrence_weekdays TEXT,
    priority INTEGER,
    energy_band TEXT,
    active BOOLEAN,
    last_done TEXT,
    success_streak INTEGER,
    avg_actual_duration REAL
);

CREATE TABLE IF NOT EXISTS plans (
    date TEXT PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS slots (
    id SERIAL PRIMARY KEY,
    plan_date TEXT,
    start_time TEXT,
    end_time TEXT,
    task_id TEXT,
    status TEXT,
    feedback_rating TEXT,
    feedback_note TEXT,
    FOREIGN KEY(plan_date) REFERENCES plans(date),
    FOREIGN KEY(task_id) REFERENCES tasks(id)
);
