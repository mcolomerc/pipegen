# Getting Started

PipeGen makes it incredibly easy to get started with streaming data pipelines. Follow this guide to create your first pipeline in minutes.

## Prerequisites

- **Docker & Docker Compose** - For local development stack
- **Go 1.21+** (optional) - Only if building from source

## Installation

### Quick Install (Recommended)

```bash
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash
```

### Alternative Methods

::: details Other installation options

**Using Go Install**
```bash
go install github.com/mcolomerc/pipegen@latest
```

**From Source**
```bash
git clone https://github.com/mcolomerc/pipegen.git
cd pipegen
go build -o pipegen ./cmd/main.go
```

**Manual Download**
1. Download the appropriate binary for your platform from the [releases page](https://github.com/mcolomerc/pipegen/releases)
2. Extract and move to your PATH

:::

## Your First Pipeline

### Step 1: Create a New Project

```bash
# Create a basic pipeline project
pipegen init my-first-pipeline

# Or use AI to generate from description
pipegen init user-analytics --describe "Track user page views and calculate session duration analytics" --domain "ecommerce"

# Initialize from a CSV file (schema inferred, filesystem source table)
pipegen init web-events --input-csv ./data/web_events_sample.csv
```

This creates a complete project structure:
```
my-first-pipeline/
├── sql/                    # FlinkSQL statements
│   ├── local/             # Local development
│   └── cloud/             # Cloud-ready templates  
├── schemas/               # AVRO schemas
│   ├── input.avsc         # Canonical input schema
│   └── output_result.avsc # AI path output schema
├── config/                # Configuration files
├── docker/               # Docker Compose setup
├── .pipegen.yaml        # Project configuration
└── README.md            # Generated documentation
```

### Step 2: Deploy Local Development Stack

```bash
cd my-first-pipeline
pipegen deploy
```

This starts:
- **Apache Kafka** (localhost:9093)
- **Apache Flink** (localhost:8081)
- **Schema Registry** (localhost:8082)
- **Flink SQL Gateway** (localhost:8083)

### Step 3: Run Your Pipeline

```bash
# Basic execution (generates automatic report)
pipegen run

# With custom report directory
pipegen run --reports-dir ./execution-reports

# With traffic pattern simulation
pipegen run --message-rate 100 --duration 5m \
  --traffic-pattern "1m-2m:300%,3m-4m:200%"

# Quick validation run
pipegen run --duration 1m --message-rate 10

# Smart consumer stopping (stops early when expected messages consumed)
pipegen run --expected-messages 500 --message-rate 50

# CSV-backed project (producer skipped automatically)
pipegen run --project-dir ./web-events
```

### Step 4: Monitor & Analyze

- **[Execution Reports](./features/reports.md)** - Comprehensive HTML reports with charts and metrics
- **Flink Web UI**: `http://localhost:8081` - Monitor jobs, checkpoints, and metrics
- **Schema Registry**: `http://localhost:8082` - Manage AVRO schemas
- **Flink SQL Gateway**: `http://localhost:8083` - SQL Gateway REST API

#### Docker Services Overview
The local development stack includes these services and ports:

| Service | Container Port | Host Port | Purpose |
|---------|----------------|-----------|---------|
| **Kafka Broker** | 9092 | 9093 | Message streaming |
| **Flink JobManager** | 8081 | 8081 | Cluster coordination & Web UI |
| **Schema Registry** | 8082 | 8082 | AVRO schema management |
| **Flink SQL Gateway** | 8083 | 8083 | SQL API endpoint |

**Connection URLs for applications:**
- Kafka: `localhost:9093`
- Flink: `http://localhost:8081`
- Schema Registry: `http://localhost:8082`
- SQL Gateway: `http://localhost:8083`

## What Happens During Execution?

1. **🔍 Project Validation** - Checks SQL syntax, schema format, and configuration
2. **🏷️ Dynamic Naming** - Generates unique topic names to avoid conflicts
3. **📝 Topic Creation** - Creates Kafka topics based on your schemas
4. **📋 Schema Registration** - Registers AVRO schemas with Schema Registry
5. **⚡ FlinkSQL Deployment** - Deploys your SQL statements as Flink jobs
6. **📤 Producer Start** - Begins generating realistic test data with AVRO encoding
7. **👂 Consumer Start** - Validates pipeline output with schema registry integration
8. **📊 Enhanced Monitoring** - Real-time metrics with consumer group lag analysis
9. **📄 Report Generation** - Creates detailed HTML execution reports (default)
10. **🧹 Cleanup** - Removes all created resources (configurable)

## Common Patterns

### Load Testing
```bash
# Simulate Black Friday traffic
pipegen run --message-rate 50 --duration 10m \
  --traffic-pattern "2m-4m:500%,6m-8m:300%,9m-10m:600%"
```

### Development Testing
```bash
# Quick validation with cleanup
pipegen run --message-rate 10 --duration 1m --cleanup=true

# Test exact message count processing
pipegen run --expected-messages 100 --message-rate 20 --timeout 5m
```

### Production Readiness
```bash
# Generate comprehensive report with separate timeouts
pipegen run --duration 3m --timeout 15m --message-rate 100 \
  --generate-report=true --reports-dir ./my-reports
```

### AVRO Schema Testing
```bash
# Test AVRO encoding with schema registry
pipegen run --message-rate 20 --duration 2m --cleanup=false
```

## Next Steps

- **[Run Workflow Deep Dive](./run-workflow)** - Detailed execution process and troubleshooting
- **[Commands](./commands)** - Learn all available commands
- **[Traffic Patterns](./traffic-patterns)** - Master load testing
- **[Dashboard](./dashboard)** - Explore monitoring capabilities
- **[AI Generation](./ai-generation)** - Use AI to generate pipelines
- **[Examples](./examples)** - See real-world use cases

## Need Help?

- **[Troubleshooting](./advanced/troubleshooting)** - Common issues and solutions
- **[GitHub Issues](https://github.com/mcolomerc/pipegen/issues)** - Report bugs or request features
- **[Configuration](./configuration)** - Advanced configuration options

::: tip Pro Tip
Use `pipegen run --dry-run` to see exactly what will be executed without actually running the pipeline. Perfect for validation!
:::
