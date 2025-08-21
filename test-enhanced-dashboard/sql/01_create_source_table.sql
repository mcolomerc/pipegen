-- Create source table for input events
CREATE TABLE input_events (
  id VARCHAR(50),
  event_type VARCHAR(100),
  user_id VARCHAR(50),
  timestamp_col TIMESTAMP(3),
  properties MAP<STRING, STRING>,
  WATERMARK FOR timestamp_col AS timestamp_col - INTERVAL '5' SECOND
) WITH (
  'connector' = 'kafka',
  'topic' = 'input-events',
  'properties.bootstrap.servers' = 'localhost:9092',
  'format' = 'avro-confluent',
  'avro-confluent.url' = 'http://localhost:8082'
);