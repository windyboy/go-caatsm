-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Ensure schema exists
CREATE SCHEMA IF NOT EXISTS aviation;

-- Create telegrams table (if not exists)
CREATE TABLE IF NOT EXISTS aviation.telegrams (
    uuid UUID PRIMARY KEY,
    message_id VARCHAR(255),
    date_time VARCHAR(255),
    priority_indicator VARCHAR(255),
    primary_address VARCHAR(255),
    secondary_addresses TEXT,
    originator VARCHAR(255),
    originator_date_time VARCHAR(255),
    category VARCHAR(255),
    content TEXT,
    body_data JSONB,
    received_at TIMESTAMPTZ NOT NULL,
    parsed_at TIMESTAMPTZ,
    dispatched_at TIMESTAMPTZ,
    need_dispatch BOOLEAN DEFAULT false
);

-- Convert to hypertable
SELECT create_hypertable(
    'aviation.telegrams',
    'received_at',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_telegrams_message_id ON aviation.telegrams (message_id);
CREATE INDEX IF NOT EXISTS idx_telegrams_date_time ON aviation.telegrams (date_time);
CREATE INDEX IF NOT EXISTS idx_telegrams_priority_indicator ON aviation.telegrams (priority_indicator);
CREATE INDEX IF NOT EXISTS idx_telegrams_primary_address ON aviation.telegrams (primary_address);
CREATE INDEX IF NOT EXISTS idx_telegrams_category ON aviation.telegrams (category);
CREATE INDEX IF NOT EXISTS idx_telegrams_received_at ON aviation.telegrams (received_at);

-- Add compression policy (compress data older than 7 days)
SELECT add_compression_policy(
    'aviation.telegrams',
    INTERVAL '7 days',
    if_not_exists => TRUE
);

-- Add retention policy (optional: remove data older than 2 years)
-- Uncomment if needed:
-- SELECT add_retention_policy(
--     'aviation.telegrams',
--     INTERVAL '2 years',
--     if_not_exists => TRUE
-- );

