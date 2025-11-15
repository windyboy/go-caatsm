CREATE SCHEMA IF NOT EXISTS aviation;

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE aviation.telegrams (
    uuid UUID NOT NULL,
    message_id TEXT,
    date_time TEXT,
    priority_indicator TEXT,
    primary_address TEXT,
    secondary_addresses TEXT,
    originator TEXT,
    originator_date_time TEXT,
    category TEXT,
    content TEXT,
    body_data JSONB,  
    status TEXT NOT NULL DEFAULT 'parsed',
    error_reason TEXT,
    received_at TIMESTAMPTZ NOT NULL,
    parsed_at TIMESTAMPTZ,
    dispatched_at TIMESTAMPTZ,
    need_dispatch BOOLEAN,
    PRIMARY KEY (uuid, received_at),
    CONSTRAINT telegrams_uuid_unique UNIQUE (uuid)
);

SELECT create_hypertable('aviation.telegrams', 'received_at', if_not_exists => TRUE);

-- Indexes for better query performance
CREATE INDEX idx_telegrams_message_id ON aviation.telegrams (message_id);
CREATE INDEX idx_telegrams_date_time ON aviation.telegrams (date_time);
CREATE INDEX idx_telegrams_priority_indicator ON aviation.telegrams (priority_indicator);
CREATE INDEX idx_telegrams_primary_address ON aviation.telegrams (primary_address);
CREATE INDEX idx_telegrams_received_at ON aviation.telegrams (received_at);
CREATE INDEX idx_telegrams_uuid ON aviation.telegrams (uuid);

CREATE TABLE IF NOT EXISTS aviation.telegrams_raw (
    uuid UUID NOT NULL,
    status TEXT NOT NULL,
    error_reason TEXT,
    content TEXT NOT NULL,
    received_at TIMESTAMPTZ NOT NULL,
    metadata JSONB,
    PRIMARY KEY (uuid, received_at),
    CONSTRAINT telegrams_raw_uuid_unique UNIQUE (uuid)
);
