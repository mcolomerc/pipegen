-- Create output table for results with AVRO format
CREATE TABLE revenue (
  `name` STRING,
  `total` INT
) WITH (
  'connector' = 'kafka',
  'topic' = 'output-results',
  'properties.bootstrap.servers' = 'broker:29092',
  'format' = 'avro-confluent',
  'avro-confluent.url' = 'http://schema-registry:8082'
);