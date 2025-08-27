-- Create output table for results
CREATE TABLE output_results (
  event_type STRING,
  user_id STRING,
  event_count BIGINT,
  window_start TIMESTAMP(3),
  window_end TIMESTAMP(3)
) WITH (
  'connector' = 'kafka',
  'topic' = 'output-results',
  'properties.bootstrap.servers' = 'localhost:9092',
  'format' = 'avro',
  'avro-confluent.schema-registry.url' = 'http://localhost:8082',
  'scan.startup.mode' = 'earliest-offset',
  'value.format' = 'avro'
);
