-- Create source table for input events (Confluent Cloud)
CREATE TABLE input_events (
  id VARCHAR(50),
  event_type VARCHAR(100),
  user_id VARCHAR(50),
  timestamp_col TIMESTAMP(3),
  properties MAP<STRING, STRING>,
  WATERMARK FOR timestamp_col AS timestamp_col - INTERVAL '5' SECOND
) WITH (
  'connector' = 'kafka',
  'topic' = '${INPUT_TOPIC}',
  'properties.bootstrap.servers' = '${BOOTSTRAP_SERVERS}',
  'properties.sasl.mechanism' = 'PLAIN',
  'properties.security.protocol' = 'SASL_SSL',
  'properties.sasl.jaas.config' = 'org.apache.kafka.common.security.plain.PlainLoginModule required username="${API_KEY}" password="${API_SECRET}";',
  'format' = 'avro-confluent',
  'avro-confluent.url' = '${SCHEMA_REGISTRY_URL}',
  'avro-confluent.basic-auth.credentials-source' = 'USER_INFO',
  'avro-confluent.basic-auth.user-info' = '${SCHEMA_REGISTRY_KEY}:${SCHEMA_REGISTRY_SECRET}'
);
