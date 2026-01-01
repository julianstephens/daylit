-- Migration 006: Consolidate OT settings into main settings table
-- Migrates OT configuration from dedicated ot_settings table into the settings key-value store

-- Migrate existing OT settings to the settings table with ot_ prefix
INSERT OR IGNORE INTO settings (key, value)
SELECT 'ot_prompt_on_empty', CASE WHEN prompt_on_empty = 1 THEN 'true' ELSE 'false' END
FROM ot_settings WHERE id = 1;

INSERT OR IGNORE INTO settings (key, value)
SELECT 'ot_strict_mode', CASE WHEN strict_mode = 1 THEN 'true' ELSE 'false' END
FROM ot_settings WHERE id = 1;

INSERT OR IGNORE INTO settings (key, value)
SELECT 'ot_default_log_days', CAST(default_log_days AS TEXT)
FROM ot_settings WHERE id = 1;

-- If ot_settings table is empty, seed with defaults
INSERT OR IGNORE INTO settings (key, value)
SELECT 'ot_prompt_on_empty', 'true'
WHERE NOT EXISTS (SELECT 1 FROM ot_settings WHERE id = 1);

INSERT OR IGNORE INTO settings (key, value)
SELECT 'ot_strict_mode', 'true'
WHERE NOT EXISTS (SELECT 1 FROM ot_settings WHERE id = 1);

INSERT OR IGNORE INTO settings (key, value)
SELECT 'ot_default_log_days', '14'
WHERE NOT EXISTS (SELECT 1 FROM ot_settings WHERE id = 1);

-- Drop the ot_settings table
DROP TABLE IF EXISTS ot_settings;
