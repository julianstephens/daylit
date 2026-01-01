-- Migration 003: Add plan revision support
-- Adds revision tracking and accepted_at timestamp to plans table
-- Changes primary key to (date, revision) composite key

-- Disable foreign key constraints during migration
PRAGMA foreign_keys=OFF;

-- First, create a new table with the correct schema
CREATE TABLE plans_new (
    date TEXT NOT NULL,
    revision INTEGER NOT NULL DEFAULT 1,
    accepted_at TEXT NULL,
    deleted_at TEXT NULL,
    PRIMARY KEY (date, revision)
);

-- Copy existing data, assigning revision 1 to all existing plans
INSERT INTO plans_new (date, revision, accepted_at, deleted_at)
SELECT date, 1, NULL, deleted_at FROM plans;

-- Update slots table to include revision
-- Create new slots table with foreign keys
CREATE TABLE slots_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plan_date TEXT NOT NULL,
    plan_revision INTEGER NOT NULL DEFAULT 1,
    start_time TEXT,
    end_time TEXT,
    task_id TEXT,
    status TEXT,
    feedback_rating TEXT,
    feedback_note TEXT,
    deleted_at TEXT NULL,
    FOREIGN KEY(plan_date, plan_revision) REFERENCES plans_new(date, revision),
    FOREIGN KEY(task_id) REFERENCES tasks(id)
);

-- Copy existing slots data, assigning revision 1
-- Note: SQLite's AUTOINCREMENT will track the highest id value and continue from max(id)+1
INSERT INTO slots_new (id, plan_date, plan_revision, start_time, end_time, task_id, status, feedback_rating, feedback_note, deleted_at)
SELECT id, plan_date, 1, start_time, end_time, task_id, status, feedback_rating, feedback_note, deleted_at FROM slots;

-- Drop old tables and rename new ones
DROP TABLE slots;
DROP TABLE plans;
ALTER TABLE plans_new RENAME TO plans;
ALTER TABLE slots_new RENAME TO slots;

-- Re-enable foreign key constraints
PRAGMA foreign_keys=ON;
