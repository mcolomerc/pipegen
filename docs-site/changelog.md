# Changelog

All notable changes to PipeGen will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-12-XX

### Added
- **Dynamic Traffic Patterns**: Simulate realistic traffic spikes and load patterns
  - Define baseline message rates with peak traffic at specific times
  - Configurable pattern syntax: `peak_percentage:start_time-end_time`
  - Multiple traffic peaks support
  - Pattern validation and conflict detection
  - Dry-run output showing traffic pattern summary

- **AI-Powered Pipeline Generation**: Natural language to FlinkSQL conversion
  - Integration with OpenAI and Ollama
  - Automatic schema generation from descriptions
  - SQL statement optimization
  - Configuration suggestions

- **Real-time Dashboard**: Live monitoring and visualization
  - WebSocket-based real-time updates  
  - Pipeline execution metrics
  - Interactive charts and graphs
  - Performance analytics

- **Comprehensive Project Scaffolding**:
  - Complete pipeline project generation
  - AVRO schema templates
  - Docker Compose setup for local development
  - Environment-specific configurations

- **Dynamic Resource Management**:
  - Automatic topic naming with timestamps
  - Schema registration and versioning
  - Environment conflict avoidance
  - Cleanup utilities

- **Validation Framework**:
  - Project structure validation
  - SQL syntax checking
  - AVRO schema validation
  - Connectivity testing

- **Execution Reports**:
  - HTML report generation
  - Performance metrics
  - Pipeline visualization
  - Export capabilities

### Features
- Support for Apache Kafka message streaming
- Apache Flink SQL processing
- Confluent Schema Registry integration
- Docker-based local development environment
- Cross-platform CLI tool (Linux, macOS, Windows)

### Documentation
- Comprehensive documentation site with VitePress
- Interactive examples and tutorials
- Command reference
- Configuration guides
- Troubleshooting documentation

---

## Installation

```bash
# Download the latest release
curl -L https://github.com/mcolomerc/pipegen/releases/latest/download/pipegen-linux -o pipegen
chmod +x pipegen
sudo mv pipegen /usr/local/bin/
```

## Getting Started

```bash
# Initialize a new pipeline project
pipegen init my-pipeline

# Generate with AI assistance
pipegen init my-ai-pipeline --llm "Process user events for analytics"

# Run with traffic patterns
pipegen run --traffic-pattern "200:09:00-10:00,300:17:00-18:00"

# Validate project
pipegen validate

# Start monitoring dashboard
pipegen dashboard
```

For more information, visit the [Getting Started Guide](/getting-started).
