CREATE TABLE co2 (
    co2 BIGINT NOT NULL,
    humidity NUMERIC NOT NULL,
    temperature NUMERIC NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL
);

SELECT create_hypertable('co2', 'timestamp');
