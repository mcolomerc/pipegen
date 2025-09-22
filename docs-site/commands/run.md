# pipegen run

The `run` command executes your streaming data pipeline with the specified configuration.

## Usage

```bash
pipegen run [flags]
```

## Examples

```bash
# Run with default configuration
pipegen run

# Run with custom config file
pipegen run --config my-config.yaml

# Run with traffic pattern simulation
pipegen run --traffic-pattern "30s-60s:300%,90s-120s:200%"

# Dry run to validate configuration
pipegen run --dry-run

# Run with custom message rate and duration
pipegen run --message-rate 50 --duration 10m

# Run with smart consumer stopping (stop after 1000 messages)
pipegen run --expected-messages 1000

# Run with pipeline timeout (separate from producer duration)
pipegen run --duration 30s --pipeline-timeout 10m

# Run with custom reports directory
pipegen run --reports-dir ./execution-reports

# Run with no cleanup (preserves execution state)
pipegen run --cleanup=false

# Run a CSV-backed pipeline (auto-detected; producer skipped, consumer runs)
pipegen run --project-dir ./web-events
```

## Flags

- `--config` - Path to configuration file (default: `.pipegen.yaml`)
- `--traffic-pattern` - Define dynamic message rates with peaks and valleys
- `--dry-run` - Validate configuration without executing
- `--message-rate` - Messages per second (default: 100)
- `--duration` - Producer execution duration (default: 30s)
- `--pipeline-timeout` - Overall pipeline timeout (default: 5m0s)
- `--expected-messages` - Expected number of messages to consume before stopping (default: 0 = auto-calculate)
- `--cleanup` - Clean up resources after execution (default: true)
- `--generate-report` - Generate HTML execution report (default: true)
- `--global-tables` - Use global table creation mode (reuse session)
- `--project-dir` - Project directory path (default: ".")
- `--reports-dir` - Directory to save execution reports (default: "./reports")
- `--help` - Show help for run command

## CSV Mode (Filesystem Source)

PipeGen supports pipelines whose source is a static or append-only CSV file via the Flink `filesystem` connector. When a project was initialized with `--input-csv`, the generated `sql/01_create_source_table.sql` contains:

```
'connector' = 'filesystem'
'path' = '...your file...'
'format' = 'csv'
```

The `run` command auto-detects this pattern and enables **CSV mode**:

- Skips the Kafka producer (data originates from the CSV file)
- Still starts the Kafka consumer to validate downstream topic output
- Monitors Flink job metrics as usual
- No extra flags required—detection is based on the source table definition

### When to Use
- Rapid prototyping with historical/batch data
- Validating transformations before wiring live Kafka ingestion
- Reproducing issues with a fixed dataset

### Behavior Differences vs Kafka Mode
| Aspect | Kafka Mode | CSV Mode |
|--------|------------|----------|
| Source | Live synthetic producer | Filesystem CSV connector |
| Producer | Started (rate driven) | Skipped |
| Consumer | Started | Started |
| Expected Messages | Auto-derived from producer or flag | Must be supplied manually if you want bounded consumption (otherwise timeout logic applies) |
| Data Volume | Controlled by duration * rate | CSV file size / row count |

### Example
```bash
pipegen init web-events --input-csv ./data/web_events.csv
pipegen deploy
pipegen run --project-dir ./web-events
```

Optional: set `--expected-messages` to the approximate row count if you want early termination after full ingestion.

## Traffic Patterns

The `--traffic-pattern` flag allows you to simulate realistic traffic with varying load:

### Format
```
"start1-end1:rate1%,start2-end2:rate2%"
```

- `start`/`end`: Time offsets (e.g., `30s`, `2m`, `1h`)
- `rate`: Percentage of baseline rate (e.g., `300%` = 3x baseline)

### Examples

```bash
# Single peak at 300% rate from 30s to 60s
pipegen run --traffic-pattern "30s-60s:300%"

# Multiple peaks: 300% for 30-60s, 200% for 90-120s
pipegen run --traffic-pattern "30s-60s:300%,90s-120s:200%"

# Black Friday simulation
pipegen run --message-rate 50 --duration 10m \
  --traffic-pattern "2m-4m:500%,6m-8m:400%,9m-10m:600%"
```

### Examples

```bash
# Simple peak pattern
pipegen run --traffic-pattern "0:10,30:100,60:10"

# Complex daily pattern
pipegen run --traffic-pattern "0:5,300:50,600:200,900:500,1200:800,1500:300,1800:100,2100:20"

# Flash sale simulation
pipegen run --traffic-pattern "0:100,60:100,61:2000,65:2000,66:100,120:100"
```

## Smart Consumer Stopping

PipeGen now features intelligent consumer stopping that automatically terminates the pipeline when all expected messages have been consumed, eliminating the need to wait for timeouts.

### How It Works

1. **Auto-calculation**: By default, PipeGen calculates expected messages based on producer output
2. **Manual override**: Use `--expected-messages` to specify exact count
3. **Smart timeout**: Consumer stops if no messages received for 30 seconds
4. **Progress tracking**: Real-time progress updates show completion percentage

### Examples

```bash
# Auto-calculate expected messages from producer output
pipegen run --duration 30s
# Producer sends 2847 messages → Consumer expects 2847 messages

# Manually specify expected message count
pipegen run --expected-messages 1000
# Consumer stops immediately after consuming 1000 messages

# Pipeline timeout vs producer duration
pipegen run --duration 10s --pipeline-timeout 5m
# Producer runs for 10s, but pipeline has 5m to complete processing
```

### Benefits

- **Faster execution**: No waiting for arbitrary timeouts
- **Precise control**: Stop exactly when work is complete
- **Better monitoring**: Real-time progress tracking
- **Robust timing**: Separate producer duration from overall pipeline timeout

### Output Example

```
📊 Expecting 908 messages based on producer output
👂 Starting consumer for topic: output-results (expecting 908 messages)
📊 Consumer progress: 454/908 messages (50.0% complete)
📊 Consumer progress: 908/908 messages (100.0% complete)
✅ Consumer completed successfully! Consumed 908/908 expected messages
```

## Configuration File

The run command uses a YAML configuration file:

```yaml
pipeline:
  name: "my-pipeline"
  schema: "schemas/input.avsc"

kafka:
  brokers:
    - "localhost:9093"
  topic: "input-topic"

flink:
  checkpoint_interval: "30s"
  parallelism: 4
```

## Validation

Before execution, `pipegen run` validates:

- Configuration file syntax
- Schema files existence and validity
- Kafka connectivity
- Flink cluster availability
- SQL syntax

## Execution Reports

Every `pipegen run` execution automatically generates comprehensive HTML reports:

### Report Generation
- **Automatic creation** for every execution
- **Saved to `reports/` folder** by default
- **Timestamped filenames** for easy identification
- **Professional styling** ready for sharing

### Report Contents
- **Executive summary** with key metrics
- **Performance charts** using Chart.js
- **Configuration snapshot** for reproducibility
- **Traffic pattern analysis** (when applicable)
- **System health monitoring** and resource usage

### Custom Report Location
```bash
# Save reports to custom directory
pipegen run --reports-dir ./my-reports

# Reports automatically named: pipegen-execution-report-YYYYMMDD-HHMMSS.html
```

## Error Handling

Common issues and solutions:

### Configuration Errors
```
Error: config file not found
Solution: Specify --config path or create config.yaml
```

### Connectivity Issues
```
Error: cannot connect to Kafka
Solution: Verify broker addresses and network connectivity
```

### Schema Validation Errors
```
Error: invalid schema format
Solution: Check schema file syntax and structure
```

## Performance Tips

1. **Parallelism**: Adjust Flink parallelism based on your cluster size
2. **Batch Size**: Configure appropriate batch sizes for your throughput
3. **Checkpointing**: Set checkpoint intervals based on your latency requirements
4. **Memory**: Allocate sufficient memory for large datasets

## See Also

- **[Run Workflow Deep Dive](../run-workflow.md)** - Complete technical workflow explanation
- **[Execution Reports](../features/reports.md)** - Detailed report documentation
- [Traffic Patterns](../traffic-patterns.md) - Understanding load testing patterns
- [Configuration](../configuration.md) - Pipeline configuration options
- [pipegen validate](./validate.md) - Pre-execution validation
