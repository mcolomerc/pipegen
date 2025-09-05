-- Create source table for Kafka stream with AVRO format
CREATE TABLE transactions_v4 (
  `name` STRING,
  `amount` INT
) WITH (
  'connector' = 'kafka',
  'topic' = 'transactions',
  'properties.bootstrap.servers' = 'broker:29092',
  'properties.group.id' = 'flink_table_transactions_v4',
  'scan.startup.mode' = 'earliest-offset',
  'properties.auto.offset.reset' = 'earliest',
  'properties.enable.auto.commit' = 'true',
  'format' = 'avro-confluent',
  'avro-confluent.url' = 'http://schema-registry:8082'
);

