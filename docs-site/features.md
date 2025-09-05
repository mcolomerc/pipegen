# Features

PipeGen provides a comprehensive set of features for creating, managing, and monitoring streaming data pipelines.

## <i class="fas fa-rocket"></i> Quick Project Scaffolding

Generate complete pipeline projects with minimal configuration. PipeGen creates:
- FlinkSQL statements for data processing
- AVRO schemas for data serialization
- Docker Compose setup for local development
- Configuration files for different environments

[Learn more about getting started →](/getting-started)

## <i class="fas fa-brain"></i> AI-Powered Generation

Describe your data pipeline in natural language and let AI handle the complexity:
- Generate optimized FlinkSQL statements
- Create appropriate AVRO schemas
- Suggest configuration parameters
- Provide implementation best practices

[Explore AI generation →](/ai-generation)

## <i class="fas fa-chart-line"></i> Dynamic Traffic Patterns

Simulate realistic production scenarios with customizable traffic patterns:
- Define baseline message rates
- Create traffic spikes at specific times
- Test pipeline performance under load
- Validate capacity planning decisions

[Configure traffic patterns →](/traffic-patterns)

## <i class="fas fa-laptop-code"></i> Local Development Stack

Complete Docker-based development environment:
- Apache Kafka for message streaming
- Apache Flink for stream processing
- Confluent Schema Registry for schema management
- Pre-configured networking and volumes

[Set up local environment →](/getting-started#local-setup)

## <i class="fas fa-code"></i> AVRO Schema Registry Integration

Full-featured schema management with automatic format detection:
- **Smart Producer**: Automatically uses AVRO format when Schema Registry is available
- **Confluent Wire Format**: Proper magic bytes and schema ID encoding
- **JSON Fallback**: Seamless fallback to JSON when no schema registry
- **Schema Evolution**: Version management and compatibility checking
- **Enhanced Monitoring**: Consumer group lag analysis for processing detection

[Learn about schema management →](/configuration#schema-configuration)

## <i class="fas fa-chart-bar"></i> Real-time Monitoring

Comprehensive execution tracking with detailed reporting:
- Automatic HTML report generation for every run
- Performance analytics and interactive charts
- Pipeline execution metrics and analysis
- Professional reports ready for sharing

[View report examples →](/features/reports)

## <i class="fas fa-cogs"></i> Dynamic Resource Management

Intelligent resource handling to avoid conflicts:
- Automatic topic naming with timestamps
- Schema registration and versioning
- Cleanup utilities for development environments
- Environment-specific configurations

[Configure resources →](/configuration)

## <i class="fas fa-check-circle"></i> Smart Consumer Stopping

Intelligent pipeline completion with automatic consumer termination:
- **Auto-calculation**: Automatically determines expected message count from producer output
- **Manual Control**: Override with `--expected-messages` for precise control
- **Progress Tracking**: Real-time progress updates with completion percentage
- **Smart Timeout**: 30-second timeout prevents hanging when no messages available
- **Separate Timeouts**: Producer duration independent of overall pipeline timeout

Benefits:
- **Faster Execution**: No waiting for arbitrary 5-minute timeouts
- **Precise Control**: Stop exactly when work is complete
- **Better UX**: Clear progress indication and immediate completion

[Learn about pipeline timing →](/commands/run#smart-consumer-stopping)

## <i class="fas fa-cogs"></i> Comprehensive Validation

Validate your pipeline before deployment:
- Project structure verification
- SQL syntax checking
- AVRO schema validation
- Connectivity testing
- Dependency verification

[Run validations →](/commands/validate)

## <i class="fas fa-file-alt"></i> Execution Reports

Generate comprehensive reports for analysis and sharing:
- Professional HTML reports with interactive charts
- Complete pipeline execution metrics and performance data
- Automatic generation with timestamped filenames saved to `reports/` folder
- Configuration snapshots and resource utilization tracking
- Easy sharing capabilities for stakeholders and team collaboration

[Learn about reports →](/features/reports)

---

## Getting Started

Ready to explore these features? [Get started with PipeGen →](/getting-started)

Or jump directly to:
- [Installation guide →](/installation)
- [Command reference →](/commands)
- [Configuration options →](/configuration)
- [Examples and tutorials →](/examples)
