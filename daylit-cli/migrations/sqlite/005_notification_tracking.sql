-- Migration 005: Add notification tracking to slots
-- Adds last_notified_start and last_notified_end timestamps to track sent notifications

ALTER TABLE slots ADD COLUMN last_notified_start TEXT NULL;
ALTER TABLE slots ADD COLUMN last_notified_end TEXT NULL;
