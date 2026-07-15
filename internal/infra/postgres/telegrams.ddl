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
    PRIMARY KEY (uuid, received_at)
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
    PRIMARY KEY (uuid, received_at)
);

-- Business-message deduplication gate.
-- aviation.telegrams is a TimescaleDB hypertable partitioned by received_at, and
-- TimescaleDB requires every unique index on a hypertable to include the
-- partitioning column. A unique index on (message_id, date_time) alone is
-- therefore invalid there. This is a plain (non-hypertable) table so it can carry
-- a primary key on the business identity (message_id, date_time).
-- Repository.InsertOne reserves a row here inside a transaction; a primary-key
-- violation means the same business message was already persisted (possibly by a
-- concurrent writer), so the telegram insert is skipped. The reservation and the
-- telegram insert share the transaction, so a failure rolls both back.
CREATE TABLE IF NOT EXISTS aviation.telegram_keys (
    message_id TEXT NOT NULL,
    date_time TEXT NOT NULL,
    PRIMARY KEY (message_id, date_time)
);
