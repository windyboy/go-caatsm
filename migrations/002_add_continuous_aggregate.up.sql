-- Create continuous aggregate view for hourly message statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS aviation.telegrams_hourly
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 hour', received_at) AS hour,
    category,
    COUNT(*) AS message_count,
    COUNT(*) FILTER (WHERE category IS NOT NULL AND category != '') AS parsed_count,
    COUNT(*) FILTER (WHERE category IS NULL OR category = '') AS unparsed_count
FROM aviation.telegrams
GROUP BY hour, category;

-- Create index on the continuous aggregate
CREATE INDEX IF NOT EXISTS idx_telegrams_hourly_hour_category 
ON aviation.telegrams_hourly (hour, category);

-- Add automatic refresh policy
-- Refresh every hour, keeping 3 hours of data in the materialized view
SELECT add_continuous_aggregate_policy(
    'aviation.telegrams_hourly',
    start_offset => INTERVAL '3 hours',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour',
    if_not_exists => TRUE
);

-- Create daily aggregate view
CREATE MATERIALIZED VIEW IF NOT EXISTS aviation.telegrams_daily
WITH (timescaledb.continuous) AS
SELECT 
    time_bucket('1 day', received_at) AS day,
    category,
    COUNT(*) AS message_count,
    COUNT(*) FILTER (WHERE category IS NOT NULL AND category != '') AS parsed_count,
    COUNT(*) FILTER (WHERE category IS NULL OR category = '') AS unparsed_count
FROM aviation.telegrams
GROUP BY day, category;

-- Create index on the daily aggregate
CREATE INDEX IF NOT EXISTS idx_telegrams_daily_day_category 
ON aviation.telegrams_daily (day, category);

-- Add automatic refresh policy for daily aggregate
SELECT add_continuous_aggregate_policy(
    'aviation.telegrams_daily',
    start_offset => INTERVAL '3 days',
    end_offset => INTERVAL '1 day',
    schedule_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

