-- Drop hypertable (this will remove the hypertable but keep the table)
-- Note: TimescaleDB doesn't provide a direct way to convert hypertable back to regular table
-- This migration assumes you want to drop the entire table

DROP TABLE IF EXISTS aviation.telegrams CASCADE;

-- Drop schema if empty (optional)
-- DROP SCHEMA IF EXISTS aviation;

