# Changelog

All notable changes to PipeGen will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- TBD

### Changed
- TBD

### Fixed
- TBD

---

## [1.5.0] - 2025-09-22

### Added
- **🎯 CSV Input Support**: Revolutionary new way to bootstrap pipelines with real data
  - `--input-csv` flag for `pipegen init` command enables direct CSV file input
  - Intelligent schema inference with automatic type detection (int, double, boolean, string, dates)
  - Support for nullable fields and union types with null
  - Streaming CSV analysis with configurable sampling limits (up to 500 rows)
  - Progressive type widening (int → double → string) for robust inference

- **🧠 Advanced Schema Inference Engine**
  - **Smart Type Detection**: Automatically infers column types from CSV data patterns
  - **Nullable Field Handling**: Detects empty values and creates appropriate union types
  - **Sample Collection**: Retains example values for better data understanding
  - **Date/Timestamp Recognition**: Supports multiple date formats and logical types
  - **Streaming Processing**: Memory-efficient analysis for large CSV files

- **🐋 Seamless Docker Integration** 
  - Automatic Docker volume configuration for CSV files
  - Creates `./data/` directory in generated projects with proper structure
  - Copies CSV files to project structure for container access
  - Enhanced docker-compose.yml with volume mounts for all Flink containers
  - Container path mapping (`/opt/flink/data/input/`) for filesystem connectors

- **🏃‍♀️ Enhanced Runtime Support**
  - **CSV Mode Auto-Detection**: `pipegen run` automatically detects filesystem CSV source tables
  - **Smart Producer Skipping**: Bypasses Kafka producer when filesystem connector detected
  - **Downstream Validation**: Still validates Kafka output and Flink job metrics
  - **Seamless Monitoring**: Full integration with existing reporting and dashboard features

- **🤖 AI Integration Enhancement**
  - CSV analysis summary integration with LLM-powered generation
  - Real data context for more accurate AI-generated pipelines
  - Combined CSV + AI description workflows for optimal results

- **📚 Comprehensive Documentation**
  - Enhanced `init.md` with CSV input examples and workflows
  - Updated `run.md` with CSV mode explanation and behavior differences
  - New sections in feature documentation for CSV capabilities
  - Updated getting started guide with CSV-first workflow examples

### Changed
- **CSV Mode Behavior**: Only Kafka producer is skipped in CSV mode; consumer validation maintained for downstream output verification
- **Template System**: Enhanced Docker compose template with automatic volume mount generation
- **Project Structure**: CSV mode projects include dedicated `data/` directory with proper file organization

### Internal
- Added comprehensive `CSVAnalyzer` component (`internal/generator/csv_analyzer.go`) for streaming CSV profiling
- Enhanced project generator with CSV-specific schema and DDL generation capabilities
- Added `CSVMode` detection logic in `cmd/run.go` for runtime behavior switching
- Refactored pipeline runner to conditionally branch around producer startup
- Added extensive test coverage for CSV analysis and type inference
- Integrated CSV analysis with existing LLM service for enhanced AI generation

### Performance
- Memory-efficient streaming CSV analysis (configurable row sampling)
- Optimized type inference with early termination for large files
- Efficient Docker volume setup and file copying

### Backward Compatibility
- All existing functionality remains unchanged
- New flags are optional and don't affect existing workflows
- Kafka-based pipelines continue to work as before
- Docker compose templates include new volume mounts by default without breaking changes

# Changelog

All notable changes to PipeGen will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.2] - 2025-09-22

### Fixed
- **Execution Report Generation**: Embedded `execution_report.html` template directly into the binary to eliminate runtime failures when the template file isn't present beside the installed executable (observed in 1.4.1 release). The system now uses the embedded template first and falls back to disk only for development overrides.

### Internal
- Added `internal/templates/embed.go` with `//go:embed` directive.
- Updated both `ExecutionReportGenerator` and pipeline runner HTML generation to support embedded fallback.

### Verification
- All unit tests pass; report generation paths exercised locally with and without on-disk template.

## [1.4.1] - 2025-09-22

### Fixed
- **Flink SQL Deployment Robustness**: Added dual-endpoint fallback for operation result retrieval. The deployer now first queries `.../result/0` (modern SQL Gateway) and transparently falls back to legacy `.../result` if a 404 is returned.
- Improved retry logic with clear context-rich error messages when all attempts fail (includes endpoint paths tried and HTTP status codes).

### Added
- **Unit Tests**: Comprehensive tests for the new `fetchOperationResult` helper covering primary success, fallback success, transient 404 recovery, and full failure scenarios.

### Internal
- Refactored result polling into a dedicated helper to simplify `deployStatement` logic and improve maintainability.

### Verification
- All tests pass (`go test ./...`), including new Flink deployer tests.
- No public API changes; patch release is backward compatible.

## [1.2.0] - 2025-09-04

## [1.4.0] - 2025-09-22

### Added
- **Execution Report Enhancements**:
  - Enriched Kafka Topics table (partitions, replication factor, total messages, size, produce / consume rate, lag)
  - Enriched Flink Jobs table (job id, status badge, duration, parallelism, records in/out, throughput, backpressure)
  - Performance metrics section formatting improvements with consistent card/table styling
  - Pipeline footer timestamp now bound to execution metadata instead of live time call
  - ASCII pipeline diagram updated with standardized units
- **Unit Standardization**: All throughput / rate metrics now displayed as `msgs/sec` (previous inconsistency with `msg/sec`).

### Fixed
- Correct timestamp rendering in inline dashboard report template (uses `.LastUpdated` instead of `time.Now` inside template scope)
- Ensured template functions consistently handle large number formatting (K / M suffix)

### Internal
- Refactored report generation logic to clearly separate inline dashboard report vs file-based execution report generator
- Added extended topic/job metric fields to internal data structures

### Verification
- All existing tests green (`make test`), lint and formatting checks pass
- Manual HTML report generation validated (Kafka & Flink sections render enriched tables)

## [1.3.0] - 2025-09-15

### Added
- Combine `--input-schema` with `--describe` to enable schema-grounded AI generation
- New LLM flow `GeneratePipelineWithSchema` using provided AVSC/JSON
- E2E tests covering schema+AI initialization and validation

### Docs
- Consolidated command docs to one page per command (removed `init-new`)
- Fixed legacy/missing links with redirects: `/configuration/environment`, `/examples/analytics`, `/examples/index`
- Sidebar “Examples & Tutorials” now points to sections on the unified `/examples` page

### Verification
- golangci-lint: pass; go fmt/mod tidy: pass; VitePress build: pass

### Added
- **Smart Consumer Stopping**: 🎯
  - Automatic consumer termination when expected message count is reached
  - Auto-calculation of expected messages based on producer output
  - Manual override with `--expected-messages` flag for precise control
  - Real-time progress tracking with completion percentage
  - 30-second smart timeout to prevent hanging when no messages available
  - Separate `--pipeline-timeout` independent of producer `--duration`

- **Enhanced Pipeline Timing**:
  - Producer duration (`--duration`) now separate from overall pipeline timeout (`--pipeline-timeout`)
  - Default producer duration reduced to 30s for faster development cycles
  - Pipeline timeout remains 5 minutes to allow Flink processing time
  - Intelligent flow control ensures consumer and Flink have time to process

- **Improved User Experience**:
  - Much faster pipeline completion (30-45 seconds vs 5+ minutes)
  - Clear progress indicators: "908/908 messages (100% complete)"
  - Better error handling and graceful stopping
  - Professional status messages and consolidated logging

### Fixed
- **Output Schema Registration**: Fixed Flink not producing output messages by ensuring both input and output AVRO schemas are registered
- **Enhanced Flink Monitoring**: Added checks for both read AND write records to verify Flink is actually producing output
- **Consumer Hanging**: Eliminated long waits for pipeline timeout when no messages are available

## [1.1.0] - 2025-09-XX

### Added
- **Enhanced AVRO Schema Registry Integration**:
  - Smart producer with automatic format detection (AVRO when schema registry available, JSON fallback)
  - Proper Confluent wire format with magic bytes and schema IDs
  - Enhanced consumer group lag monitoring for better processing detection
  - Improved connector compatibility (resolved version conflicts)

- **Improved CLI Experience**:
  - Updated flag structure (`--message-rate`, `--duration` instead of `--rate`, `--messages`)
  - Enhanced cleanup control with `--cleanup=true/false`
  - HTML report generation enabled by default (`--generate-report`)
  - Global table creation mode (`--global-tables`)
  - Configurable dashboard port (`--dashboard-port`)

- **Better Monitoring & Reports**:
  - Enhanced monitoring with consumer group lag analysis
  - More reliable processing detection
  - Improved HTML execution reports with professional theme
  - Real-time dashboard improvements

### Fixed
- **AVRO Producer**: Fixed hardcoded JSON encoding - now properly uses AVRO format
- **Connector Issues**: Resolved Flink AVRO connector version conflicts
- **Schema Registry**: Improved schema registration and retrieval reliability

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
