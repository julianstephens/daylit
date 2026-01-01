-- Migration 006: Consolidate OT settings into main settings table
-- Migrates OT configuration from dedicated ot_settings table into the settings key-value store

-- Migrate existing OT settings to the settings table with ot_ prefix
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

-- If ot_settings table is empty, seed with defaults
INSERT INTO settings (key, value)
SELECT 'ot_prompt_on_empty', 'true'
WHERE NOT EXISTS (SELECT 1 FROM ot_settings WHERE id = 1)
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value)
SELECT 'ot_strict_mode', 'true'
WHERE NOT EXISTS (SELECT 1 FROM ot_settings WHERE id = 1)
ON CONFLICT (key) DO NOTHING;

INSERT INTO settings (key, value)
SELECT 'ot_default_log_days', '14'
WHERE NOT EXISTS (SELECT 1 FROM ot_settings WHERE id = 1)
ON CONFLICT (key) DO NOTHING;

-- Drop the ot_settings table
DROP TABLE IF EXISTS ot_settings;
