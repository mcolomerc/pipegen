-- Create output table for results (Confluent Cloud)
CREATE TABLE output_results (
  event_type STRING,
  user_id STRING,
  event_count BIGINT,
  window_start TIMESTAMP(3),
  window_end TIMESTAMP(3)
) WITH (
  'connector' = 'kafka',
  'topic' = '${OUTPUT_TOPIC}',
  'properties.bootstrap.servers' = '${BOOTSTRAP_SERVERS}',
  'properties.sasl.mechanism' = 'PLAIN',
  'properties.security.protocol' = 'SASL_SSL',
  'properties.sasl.jaas.config' = 'org.apache.kafka.common.security.plain.PlainLoginModule required username="${API_KEY}" password="${API_SECRET}";',
  'format' = 'avro-confluent',
  'avro-confluent.url' = '${SCHEMA_REGISTRY_URL}',
  'avro-confluent.basic-auth.credentials-source' = 'USER_INFO',
  'avro-confluent.basic-auth.user-info' = '${SCHEMA_REGISTRY_KEY}:${SCHEMA_REGISTRY_SECRET}'
);
