# Commands

PipeGen provides a comprehensive set of commands for creating, managing, and monitoring streaming data pipelines. Each command is designed to be intuitive and powerful.

## Command Overview

| Command | Purpose | Use Case |
|---------|---------|----------|
| [`init`](./commands/init) | Create new pipeline project | Project scaffolding, AI generation |
| [`run`](./commands/run) | Execute complete pipeline | Testing, load testing, production runs |
| [`deploy`](./commands/deploy) | Start local development stack | Local development environment |
| [`validate`](./commands/validate) | Validate project structure | Pre-deployment checks |
| [`check`](./commands/check) | Check AI provider setup | AI configuration validation |
| [`clean`](./commands/clean) | Clean up Docker resources | Free up system resources |

## Quick Reference

### Project Creation
```bash
# Basic project
pipegen init my-pipeline

# AI-generated project
pipegen init fraud-detection --describe "Monitor transactions for fraud"

# Custom schema project  
pipegen init ecommerce --input-schema ./events.avsc

# Schema + AI description (ground AI with your schema)
pipegen init orders --input-schema ./schemas/input.avsc \
  --describe "Join orders with customers, compute daily GMV and top-5 products"
```

### Pipeline Execution
```bash
# Basic run
pipegen run

# With traffic patterns
pipegen run --traffic-pattern "1m-2m:300%,4m-5m:200%"

# With automatic HTML report generation
pipegen run --reports-dir ./my-reports

# Smart consumer stopping (stops after expected messages)
pipegen run --expected-messages 1000 --duration 5m

# Separate producer and overall timeouts
pipegen run --duration 2m --timeout 10m

# Dry run (preview only)
pipegen run --dry-run
```

### Development Environment
```bash
# Deploy local stack
pipegen deploy

# Deploy with custom timeout
pipegen deploy --startup-timeout 3m

# Clean deploy (remove existing containers)
pipegen deploy --clean
```

### Monitoring & Validation
```bash
# Standalone dashboard
pipegen dashboard --standalone

# Validate project
pipegen validate --check-connectivity

# Check AI setup
pipegen check
```

### Cleaning Up
```bash
# Basic cleanup
pipegen clean

# Cleanup with volume removal
pipegen clean --volumes

# Force cleanup without prompt
pipegen clean --force
```

## Global Flags

These flags are available for all commands:

| Flag | Default | Description |
|------|---------|-------------|
| `--bootstrap-servers` | `localhost:9093` | Kafka bootstrap servers |
| `--flink-url` | `http://localhost:8081` | Flink Job Manager URL |
| `--schema-registry-url` | `http://localhost:8082` | Schema Registry URL |
| `--config` | `$HOME/.pipegen.yaml` | Configuration file path |
| `--local-mode` | `true` | Use local development mode |

### Examples with Global Flags

```bash
# Connect to cloud Kafka cluster
pipegen run --bootstrap-servers "pkc-4nym6.us-east-1.aws.confluent.cloud:9092"

# Use custom Flink cluster
pipegen run --flink-url "https://my-flink.company.com:8081"

# Use custom config file
pipegen --config ./my-config.yaml run
```

## Environment Variables

You can also set configuration using environment variables:

```bash
export PIPEGEN_BOOTSTRAP_SERVERS="localhost:9093"
export PIPEGEN_FLINK_URL="http://localhost:8081"
export PIPEGEN_SCHEMA_REGISTRY_URL="http://localhost:8082"

# For AI features
export PIPEGEN_OLLAMA_MODEL="llama3.1"
export PIPEGEN_OPENAI_API_KEY="your-key"
```

## Common Workflows

### 🚀 New Project Workflow
```bash
# 1. Create project
pipegen init my-analytics --describe "User behavior analytics"

# 2. Deploy local environment  
cd my-analytics
pipegen deploy

# 3. Validate setup
pipegen validate --check-connectivity

# 4. Run with monitoring
pipegen run --dashboard
```

### 🧪 Testing Workflow
```bash
# 1. Quick validation
pipegen run --dry-run

# 2. Basic test
pipegen run --message-rate 10 --duration 1m

# 3. Smart stopping test (stops early when done)
pipegen run --message-rate 50 --expected-messages 500 --timeout 5m

# 4. Load test with patterns
pipegen run --message-rate 100 --duration 5m \
  --traffic-pattern "1m-2m:300%,3m-4m:200%"

# 5. Generate comprehensive report
pipegen run --message-rate 50 --duration 10m \
  --dashboard --generate-report
```

### 🔍 Debugging Workflow
```bash
# 1. Check configuration
pipegen check

# 2. Validate project
pipegen validate --check-connectivity

# 3. Start monitoring dashboard
pipegen dashboard --standalone

# 4. Run with detailed logging
pipegen run --dashboard --message-rate 1 --duration 5m
```

### 🏢 Production Workflow
```bash
# 1. Generate production-ready project
pipegen init production-pipeline --describe "Real-time order processing"

# 2. Validate against cloud services
pipegen validate --check-connectivity \
  --bootstrap-servers "your-kafka-cluster:9092"

# 3. Load test with realistic patterns
pipegen run --message-rate 200 --duration 30m \
  --traffic-pattern "5m-15m:300%,20m-25m:400%" \
  --dashboard --generate-report

# 4. Deploy SQL to production Flink
# (Manual step using generated SQL files)
```

## Configuration Hierarchy

PipeGen uses the following configuration priority (highest to lowest):

1. **Command-line flags**
2. **Environment variables**  
3. **Project `.pipegen.yaml`**
4. **Global `~/.pipegen.yaml`**
5. **Default values**

### Example Configuration File
```yaml
# ~/.pipegen.yaml or .pipegen.yaml
bootstrap_servers: "localhost:9093"
flink_url: "http://localhost:8081"
schema_registry_url: "http://localhost:8082"
local_mode: true

# Default pipeline settings
default_message_rate: 100
default_duration: "5m" 
topic_prefix: "pipegen"
cleanup_on_exit: true

# AI settings
ollama_model: "llama3.1"
ollama_url: "http://localhost:11434"
```

## Command Chaining

Some commands work well together:

```bash
# Create, validate, and run
pipegen init my-pipeline --describe "IoT sensor processing" && \
cd my-pipeline && \
pipegen validate && \
pipegen deploy && \
pipegen run --dashboard

# Test multiple scenarios
pipegen run --message-rate 50 --duration 2m && \
pipegen run --message-rate 100 --duration 2m && \
pipegen run --message-rate 200 --duration 2m
```

## Error Handling

PipeGen provides clear error messages and suggestions:

```bash
# Missing configuration
❌ Error: missing required configuration: bootstrap_servers
💡 Tip: Set via --bootstrap-servers flag or PIPEGEN_BOOTSTRAP_SERVERS env var

# Invalid traffic pattern  
❌ Error: invalid traffic pattern: patterns overlap
💡 Tip: Ensure traffic patterns don't overlap in time

# Connection issues
❌ Error: failed to connect to Kafka at localhost:9093
💡 Tip: Ensure Kafka is running via 'pipegen deploy'
```

## Getting Help

Each command has built-in help:

```bash
# General help
pipegen --help

# Command-specific help
pipegen run --help
pipegen init --help  
pipegen deploy --help
```

## Next Steps

Explore detailed documentation for each command:

- **[pipegen init](./commands/init)** - Project creation and AI generation
- **[pipegen run](./commands/run)** - Pipeline execution and testing
- **[pipegen deploy](./commands/deploy)** - Local development environment  
- **[pipegen validate](./commands/validate)** - Project validation
- **[pipegen dashboard](./commands/dashboard)** - Real-time monitoring
- **[Configuration](./configuration)** - Advanced configuration options
