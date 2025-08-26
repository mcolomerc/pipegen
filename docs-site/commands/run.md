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
pipegen run --traffic-pattern "0:100,60:500,120:200,180:1000"

# Dry run to validate configuration
pipegen run --dry-run

# Run with specific message count
pipegen run --messages 1000

# Run with custom rate
pipegen run --rate 10
```

## Flags

- `--config` - Path to configuration file (default: config.yaml)
- `--traffic-pattern` - Define dynamic message rates with peaks and valleys
- `--dry-run` - Validate configuration without executing
- `--messages` - Number of messages to send (default: unlimited)
- `--rate` - Messages per second (default: 1)
- `--format` - Output format (table, json, yaml)
- `--verbose` - Enable verbose logging
- `--help` - Show help for run command

## Traffic Patterns

The `--traffic-pattern` flag allows you to simulate realistic traffic with varying load:

### Format
```
"time1:rate1,time2:rate2,time3:rate3"
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

## Configuration File

The run command uses a YAML configuration file:

```yaml
pipeline:
  name: "my-pipeline"
  schema: "schemas/input.json"
  
kafka:
  brokers:
    - "localhost:9092"
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

## Monitoring

During execution, the command provides:

- Real-time message throughput
- Processing latency metrics
- Error rates and status
- Traffic pattern visualization

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

- [Traffic Patterns](../traffic-patterns.md)
- [Configuration](../configuration.md)
- [Dashboard](../dashboard.md)
- [pipegen validate](./validate.md)
