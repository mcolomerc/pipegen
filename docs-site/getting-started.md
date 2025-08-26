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
```

This creates a complete project structure:
```
my-first-pipeline/
├── sql/                    # FlinkSQL statements
│   ├── local/             # Local development
│   └── cloud/             # Cloud-ready templates  
├── schemas/               # AVRO schemas
│   ├── input.json
│   └── output.json
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
- **Apache Kafka** (localhost:9092)
- **Apache Flink** (localhost:8081) 
- **Schema Registry** (localhost:8082)
- **Flink Web UI** (localhost:8081)

### Step 3: Run Your Pipeline

```bash
# Basic execution
pipegen run

# With live dashboard
pipegen run --dashboard

# With traffic pattern simulation
pipegen run --message-rate 100 --duration 5m --traffic-pattern "1m-2m:300%,3m-4m:200%"
```

### Step 4: Monitor & Analyze

- **Live Dashboard**: `http://localhost:3000` (if using `--dashboard`)
- **Flink Web UI**: `http://localhost:8081`
- **Execution Reports**: Generated in `reports/` directory

## What Happens During Execution?

1. **🔍 Project Validation** - Checks SQL syntax, schema format, and configuration
2. **🏷️ Dynamic Naming** - Generates unique topic names to avoid conflicts
3. **📝 Topic Creation** - Creates Kafka topics based on your schemas
4. **📋 Schema Registration** - Registers AVRO schemas with Schema Registry  
5. **⚡ FlinkSQL Deployment** - Deploys your SQL statements as Flink jobs
6. **📤 Producer Start** - Begins generating realistic test data
7. **👂 Consumer Start** - Validates pipeline output
8. **📊 Monitoring** - Real-time metrics collection and visualization
9. **🧹 Cleanup** - Removes all created resources (optional)

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
pipegen run --message-rate 10 --duration 1m --cleanup
```

### Production Readiness
```bash
# Generate comprehensive report
pipegen run --message-rate 100 --duration 30m --generate-report --dashboard
```

## Next Steps

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
