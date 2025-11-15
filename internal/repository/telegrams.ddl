CREATE SCHEMA IF NOT EXISTS aviation;

CREATE TABLE aviation.telegrams (
    uuid UUID PRIMARY KEY,
    message_id VARCHAR(255) ,
    date_time VARCHAR(255) ,
    priority_indicator VARCHAR(255) ,
    primary_address VARCHAR(255) ,
    secondary_addresses TEXT,  
    originator VARCHAR(255),
    originator_date_time VARCHAR(255),
    category VARCHAR(255),
    content TEXT,
    body_data JSONB,  
    status VARCHAR(32) NOT NULL DEFAULT 'parsed',
    error_reason TEXT,
    received_at TIMESTAMP NOT NULL,
    parsed_at TIMESTAMP,
    dispatched_at TIMESTAMP,
    need_dispatch BOOLEAN
);

-- Indexes for better query performance
CREATE INDEX idx_telegrams_message_id ON aviation.telegrams (message_id);
CREATE INDEX idx_telegrams_date_time ON aviation.telegrams (date_time);
CREATE INDEX idx_telegrams_priority_indicator ON aviation.telegrams (priority_indicator);
CREATE INDEX idx_telegrams_primary_address ON aviation.telegrams (primary_address);
CREATE INDEX idx_telegrams_received_at ON aviation.telegrams (received_at);

CREATE TABLE IF NOT EXISTS aviation.telegrams_raw (
    uuid UUID PRIMARY KEY,
    status VARCHAR(32) NOT NULL,
    error_reason TEXT,
    content TEXT NOT NULL,
    received_at TIMESTAMP NOT NULL,
    metadata JSONB
);
