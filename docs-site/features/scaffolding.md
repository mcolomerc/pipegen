# Project Scaffolding

PipeGen provides powerful project scaffolding capabilities to quickly bootstrap streaming data pipeline projects with all necessary components.

## Overview

The `pipegen init` command creates complete pipeline projects with:

- **FlinkSQL processing logic** optimized for your use case
- **AVRO schemas** for data serialization and validation
- **Docker Compose configuration** for local development
- **Configuration files** for different environments
- **Directory structure** following best practices

## Basic Project Creation

### Standard Project Structure
```bash
pipegen init my-pipeline
```

Creates a project with this structure:
```
my-pipeline/
├── docker-compose.yml          # Local development stack
├── .pipegen.yaml              # Pipeline configuration
├── schemas/
│   ├── input.avsc             # Input data schema (canonical)
│   └── output_result.avsc     # Output data schema (AI path)
├── sql/
│   ├── 01_create_source_table.sql
│   ├── 02_create_output_table.sql
│   └── 03_processing_logic.sql
├── connectors/                # Flink connector JARs
└── reports/                   # Execution reports directory
```

## AI-Powered Generation

### Natural Language Pipeline Creation
```bash
pipegen init fraud-detection \
  --describe "Monitor credit card transactions for suspicious patterns"
```

The AI generates:
- **Relevant AVRO schemas** with appropriate fields
- **Optimized FlinkSQL** for the specific use case
- **Processing logic** tailored to the domain
- **Configuration** with sensible defaults

### Custom Domain Examples
```bash
# E-commerce analytics
pipegen init ecommerce-analytics \
  --describe "Track user behavior and purchase patterns"

# IoT data processing
pipegen init sensor-monitoring \
  --describe "Process temperature and humidity sensor data"

# Financial analytics
pipegen init trading-analysis \
  --describe "Analyze stock price movements and volumes"
```

## Advanced Scaffolding Options

### Custom Schema Input
```bash
# Use existing input schema
pipegen init my-pipeline --input-schema ./existing-input.avsc
```

### Infer Schema from CSV
```bash
# Analyze CSV, infer schema & generate filesystem source table
pipegen init sessions --input-csv ./data/sessions.csv

# Combine CSV inference with AI aggregation generation
pipegen init session-aggregates \
  --input-csv ./data/sessions.csv \
  --describe "Daily active users, average session length, bounce rate" \
  --domain ecommerce
```

CSV inference produces:
- `schemas/input.avsc` (inferred types, nullability)
- `sql/01_create_source_table.sql` with `filesystem` + `csv` connector
- Column profile fed into AI prompt (when `--describe` used)

### Template Selection
```bash
# Real-time analytics template
pipegen init analytics-pipeline --template realtime-analytics

# ETL processing template
pipegen init etl-pipeline --template batch-processing

# Stream joining template
pipegen init join-pipeline --template stream-join
```

## Directory Structure Details

### Configuration Files
- **`.pipegen.yaml`** - Main pipeline configuration
- **`docker-compose.yml`** - Local development environment
- **`flink-conf.yaml`** - Flink cluster settings

### Schema Management
- **`schemas/`** - AVRO schema definitions
- **`input.avsc`** - Source data structure (canonical)
- **`output_result.avsc`** - AI-generated processed data structure (AI path)

### SQL Processing Logic
- **`sql/`** - FlinkSQL files in execution order
- **`01_create_source_table.sql`** - Input table definition
  - In CSV mode: generated with inferred columns and filesystem CSV connector
- **`02_create_output_table.sql`** - Output table definition
- **`03_processing_logic.sql`** - Data transformation logic

### Connector Libraries
- **`connectors/`** - Required Flink connector JARs
- **Auto-populated** with necessary dependencies during project initialization
- **Customizable**: Add additional connector JARs and restart containers to load them
- **Version-aligned**: All connectors are compatible with Flink 1.18.x

#### Adding Custom Connectors
```bash
# Add custom connectors to the connectors/ directory
cp my-custom-connector.jar ./connectors/

# Restart containers to load new connectors
docker-compose restart

# Or restart specific Flink services
docker-compose restart flink-jobmanager flink-taskmanager sql-gateway
```

#### Default Included Connectors
- **Kafka Connector**: `flink-sql-connector-kafka-3.1.0-1.18.jar`
- **AVRO Schema Registry**: `flink-sql-avro-confluent-registry-1.18.1.jar`
- **Jackson Libraries**: For JSON processing
- **Supporting Dependencies**: Kafka clients, Guava, etc.

### Output Directory
- **`reports/`** - Execution report storage
- Automatically created for HTML report generation

## Best Practices

### Project Naming
```bash
# Use descriptive, lowercase names
pipegen init user-activity-analysis
pipegen init payment-fraud-detection
pipegen init iot-sensor-aggregation
```

### Schema Design
- **Start simple** and iterate based on actual data
- **Use meaningful field names** and descriptions
- **Include data validation** constraints where possible
- **Version schemas** for production environments

### SQL Organization
- **Keep files focused** on single responsibilities
- **Use clear naming** that reflects execution order
- **Add comments** explaining complex transformations
- **Test locally** before deployment

## Integration with Existing Projects

### Adding to Existing Repository
```bash
# Initialize in existing directory
cd existing-project
pipegen init . --template basic

# Create subfolder for pipeline
mkdir streaming-pipeline
cd streaming-pipeline
pipegen init . --describe "Process user events"
```

### Git Integration
```bash
# Initialize with git
pipegen init my-pipeline --git

# Add to existing repository
cd my-pipeline
git add .
git commit -m "Add streaming pipeline scaffolding"
```

## Customization After Scaffolding

### Modifying Generated Code
1. **Review generated schemas** and adjust field types
2. **Customize SQL logic** for specific business rules
3. **Update configuration** for your environment
4. **Add additional transformations** as needed

### Local Testing
```bash
# Deploy local environment
pipegen deploy

# Test the pipeline
pipegen run --duration 30s

# Review generated report
open reports/pipegen-execution-report-*.html
```

## Template Library

### Available Templates
- **`basic`** - Simple input→processing→output pipeline
- **`realtime-analytics`** - Time-windowed aggregations
- **`stream-join`** - Multi-stream correlation
- **`batch-processing`** - Scheduled batch jobs
- **`fraud-detection`** - Financial anomaly detection
- **`iot-processing`** - Sensor data aggregation

### Template Customization
Templates can be customized by:
- **Modifying base templates** in the PipeGen installation
- **Creating organization-specific** template repositories
- **Sharing templates** across development teams

## See Also

- [AI-Powered Generation](../ai-generation.md) - Deep dive into AI capabilities
- [Configuration Guide](../configuration.md) - Understanding configuration options
- [Getting Started](../getting-started.md) - Complete setup walkthrough
- [pipegen init command](../commands/init.md) - Full command reference
