-- Create source table for input events
CREATE TABLE input_events (
  event_id STRING,
  event_type STRING,
  user_id STRING,
  ts TIMESTAMP(3),
  properties MAP<STRING, STRING>,
  WATERMARK FOR ts AS ts - INTERVAL '5' SECOND
) WITH ( 
  'connector' = 'kafka',
  'topic' = 'input-events',
  'properties.bootstrap.servers' = 'localhost:9092',
  'format' = 'avro',
  'avro-confluent.schema-registry.url' = 'http://localhost:8082',
  'scan.startup.mode' = 'earliest-offset',
  'value.format' = 'avro'
);

