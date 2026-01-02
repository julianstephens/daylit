-- Migration 008: Add timezone setting
-- Adds support for user-configured timezone

-- Add timezone setting with default value 'Local'
INSERT INTO settings (key, value) VALUES ('timezone', 'Local') ON CONFLICT (key) DO NOTHING;
