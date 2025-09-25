package docker

import (
	"fmt"
	"strings"
)

// DockerComposeGenerator creates docker-compose.yml content for the streaming stack
type DockerComposeGenerator struct{}

// NewDockerComposeGenerator creates a new generator instance
func NewDockerComposeGenerator() *DockerComposeGenerator {
	return &DockerComposeGenerator{}
}

// Generate creates the docker-compose.yml content
func (g *DockerComposeGenerator) Generate(withSchemaRegistry bool) (string, error) {
	var services []string

	// Add Kafka service (KRaft mode)
	services = append(services, g.generateKafkaService())

	// Add Flink services
	services = append(services, g.generateFlinkJobManager())
	// Add SQL Gateway service (REST backend)
	services = append(services, g.generateSqlGateway())
	services = append(services, g.generateFlinkTaskManager())

	// Add Schema Registry if requested
	if withSchemaRegistry {
		services = append(services, g.generateSchemaRegistry())
	}

	// Create volumes and networks
	volumes := g.generateVolumes()
	networks := g.generateNetworks()

	// Combine everything
	compose := fmt.Sprintf(`version: '3.8'

services:
%s

%s

%s
`, strings.Join(services, "\n"), volumes, networks)

	return compose, nil
}

func (g *DockerComposeGenerator) generateKafkaService() string {
	return `  kafka:
    image: confluentinc/cp-kafka:7.5.0
    hostname: kafka
    container_name: pipegen-kafka
    ports:
      - "9092:9092"
      - "19092:19092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: 'CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT'
      KAFKA_ADVERTISED_LISTENERS: 'PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092'
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_JMX_PORT: 19092
      KAFKA_JMX_HOSTNAME: localhost
      KAFKA_PROCESS_ROLES: 'broker,controller'
      KAFKA_CONTROLLER_QUORUM_VOTERS: '1@kafka:29093'
      KAFKA_LISTENERS: 'PLAINTEXT://kafka:29092,CONTROLLER://kafka:29093,PLAINTEXT_HOST://0.0.0.0:9092'
      KAFKA_INTER_BROKER_LISTENER_NAME: 'PLAINTEXT'
      KAFKA_CONTROLLER_LISTENER_NAMES: 'CONTROLLER'
      KAFKA_LOG_DIRS: '/tmp/kraft-combined-logs'
      CLUSTER_ID: 'MkU3OEVBNTcwNTJENDM2Qk'
    volumes:
      - kafka-data:/var/lib/kafka/data
    networks:
      - pipegen-network
    healthcheck:
      test: ["CMD-SHELL", "kafka-topics --bootstrap-server localhost:9092 --list"]
      interval: 10s
      timeout: 5s
      retries: 5`
}

func (g *DockerComposeGenerator) generateFlinkJobManager() string {
	return `  flink-jobmanager:
    image: flink:1.18.0-scala_2.12-java11
    hostname: flink-jobmanager
    container_name: pipegen-flink-jobmanager
    ports:
      - "8081:8081"
      - "6123:6123"
    command: jobmanager
    depends_on:
      kafka:
        condition: service_healthy
    environment:
      FLINK_PROPERTIES: "jobmanager.rpc.address: flink-jobmanager\njobmanager.rpc.port: 6123\nrest.address: flink-jobmanager\nrest.bind-port: 8081"
    volumes:
      - flink-data:/opt/flink/data
      - ./flink-conf.yaml:/opt/flink/conf/flink-conf.yaml
    networks:
      - pipegen-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
`
}

func (g *DockerComposeGenerator) generateFlinkTaskManager() string {
	return `  flink-taskmanager:
    image: flink:1.18.0-scala_2.12-java11
    hostname: flink-taskmanager
    container_name: pipegen-flink-taskmanager
    depends_on:
      flink-jobmanager:
        condition: service_healthy
    command: taskmanager
    scale: 1
    environment:
      FLINK_PROPERTIES: "jobmanager.rpc.address: flink-jobmanager"
    volumes:
      - flink-data:/opt/flink/data
      - ./flink-conf.yaml:/opt/flink/conf/flink-conf.yaml
    networks:
      - pipegen-network`
}

func (g *DockerComposeGenerator) generateSqlGateway() string {
	return `  sql-gateway:
    image: flink:1.18.0-scala_2.12-java11
    container_name: pipegen-sql-gateway
    command: >
      sh -c "\
      cp /opt/flink/lib/connectors/*.jar /opt/flink/lib/ && \
      bin/sql-gateway.sh start-foreground \
        -Dsql-gateway.endpoint.rest.address=0.0.0.0 \
        -Dsql-gateway.endpoint.rest.port=8083 \
        -Dsql-gateway.backend.type=rest \
        -Dsql-gateway.backend.rest.address=flink-jobmanager \
        -Dsql-gateway.backend.rest.port=8081\
      "
    ports:
      - "8083:8083"
    volumes:
      - ./connectors:/opt/flink/lib/connectors
      - ./flink-conf.yaml:/opt/flink/conf/flink-conf.yaml
      - ./flink-entrypoint.sh:/flink-entrypoint.sh
      - ./data:/opt/flink/data/input
    depends_on:
      flink-jobmanager:
        condition: service_healthy
    networks:
      - pipegen-network`
}

func (g *DockerComposeGenerator) generateSchemaRegistry() string {
	return `  schema-registry:
    image: confluentinc/cp-schema-registry:7.5.0
    hostname: schema-registry
    container_name: pipegen-schema-registry
    depends_on:
      kafka:
        condition: service_healthy
    ports:
      - "8082:8082"
    environment:
      SCHEMA_REGISTRY_HOST_NAME: schema-registry
      SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS: 'kafka:29092'
      SCHEMA_REGISTRY_LISTENERS: http://0.0.0.0:8082
    networks:
      - pipegen-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/"]
      interval: 10s
      timeout: 5s
      retries: 5`
}

func (g *DockerComposeGenerator) generateVolumes() string {
	return `volumes:
  kafka-data:
    driver: local
  flink-data:
    driver: local`
}

func (g *DockerComposeGenerator) generateNetworks() string {
	return `networks:
  pipegen-network:
    driver: bridge`
}

// GenerateFlinkConfig creates the Flink configuration file content
func (g *DockerComposeGenerator) GenerateFlinkConfig() string {
	return `# Flink Configuration for PipeGen
# JobManager
jobmanager.rpc.address: flink-jobmanager
jobmanager.rpc.port: 6123
jobmanager.memory.process.size: 1600m

# TaskManager
taskmanager.numberOfTaskSlots: 4
taskmanager.memory.process.size: 1728m
taskmanager.memory.flink.size: 1280m

# Parallelism
parallelism.default: 1

# Checkpointing
state.backend: filesystem
state.checkpoints.dir: file:///opt/flink/data/checkpoints
state.savepoints.dir: file:///opt/flink/data/savepoints
execution.checkpointing.interval: 60000

# Restart strategy
restart-strategy: fixed-delay
restart-strategy.fixed-delay.attempts: 3
restart-strategy.fixed-delay.delay: 10s

# Web UI
web.submit.enable: true
web.cancel.enable: true

# SQL Gateway (for SQL job submission)
sql-gateway.endpoint.rest.address: 0.0.0.0
sql-gateway.endpoint.rest.port: 8083

# Table planner
table.planner: blink

# Kafka connector
# These settings are automatically included with the Flink Kafka connector
`
}
