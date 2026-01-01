-- Migration 006: Consolidate OT settings into main settings table
-- Migrates OT configuration from dedicated ot_settings table into the settings key-value store

-- Migrate existing OT settings to the settings table with ot_ prefix (if ot_settings exists)
INSERT INTO settings (key, value)
SELECT 'ot_prompt_on_empty', CASE WHEN prompt_on_empty THEN 'true' ELSE 'false' END
FROM ot_settings WHERE id = 1
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value)
SELECT 'ot_strict_mode', CASE WHEN strict_mode THEN 'true' ELSE 'false' END
FROM ot_settings WHERE id = 1
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value)
SELECT 'ot_default_log_days', default_log_days::TEXT
FROM ot_settings WHERE id = 1
ON CONFLICT (key) DO NOTHING;

-- If ot_settings table existed but was empty, OR if it didn't exist, seed with defaults
INSERT INTO settings (key, value) VALUES ('ot_prompt_on_empty', 'true') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('ot_strict_mode', 'true') ON CONFLICT (key) DO NOTHING;
INSERT INTO settings (key, value) VALUES ('ot_default_log_days', '14') ON CONFLICT (key) DO NOTHING;

-- Drop the ot_settings table (if it exists)
DROP TABLE IF EXISTS ot_settings;
