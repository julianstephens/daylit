-- Migration 002: Add soft delete support
-- Adds deleted_at columns to tasks, plans, and slots tables

ALTER TABLE tasks ADD COLUMN deleted_at TEXT NULL;
ALTER TABLE plans ADD COLUMN deleted_at TEXT NULL;
ALTER TABLE slots ADD COLUMN deleted_at TEXT NULL;
