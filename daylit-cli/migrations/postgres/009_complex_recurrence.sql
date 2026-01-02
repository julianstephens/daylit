-- Migration 008: Add support for complex recurrence patterns
-- This migration adds fields to support monthly, yearly, and weekdays recurrence types

ALTER TABLE tasks ADD COLUMN recurrence_month_day INTEGER DEFAULT NULL;
ALTER TABLE tasks ADD COLUMN recurrence_week_occurrence INTEGER DEFAULT NULL;
ALTER TABLE tasks ADD COLUMN recurrence_month INTEGER DEFAULT NULL;
ALTER TABLE tasks ADD COLUMN recurrence_day_of_week INTEGER DEFAULT NULL;
