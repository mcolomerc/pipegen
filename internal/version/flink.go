package version

// Centralized version constants for Flink runtime and related connectors / libraries.
// Updating Flink generally requires reviewing all these constants and regeneration logic.

const (
	// Flink core version (used for image tag; include only major.minor.patch part)
	FlinkVersion = "2.1.0"

	// Scala binary version used in the docker image tag (2.12 or 2.13)
	FlinkScalaVersion = "2.12"

	// Connector compatibility segment used in connector artifact naming (e.g. -2.0 suffix)
	FlinkConnectorCompat = "2.0"

	// Kafka connector (DataStream & SQL) major line for Flink 2.x
	KafkaConnectorVersion = "4.0.1"

	// Confluent Avro SQL format jar version aligned with Flink major
	AvroConfluentRegistryVersion = "2.1.0"

	// Format libraries (JSON, CSV)
	FlinkFormatVersion = "2.1.0"

	// Supporting library versions (keep in sync with Flink dependency tree when possible)
	KafkaClientsVersion = "3.7.0"
	JacksonVersion      = "2.17.2"
	GuavaVersion        = "33.2.1-jre"
)

// Derived helper functions (if needed later) can be added here.
