# pipegen validate

The `validate` command checks your pipeline configuration, schemas, and SQL for correctness before execution.

## Usage

```bash
pipegen validate [flags]
```

## Examples

```bash
# Validate current project
pipegen validate

# Validate specific config file
pipegen validate --config my-config.yaml

# Validate with verbose output
pipegen validate --verbose

# Validate only schemas
pipegen validate --schemas-only

# Validate only SQL files
pipegen validate --sql-only
```

## Flags

- `--config` - Configuration file to validate (default: config.yaml)
- `--schemas-only` - Validate only schema files
- `--sql-only` - Validate only SQL files
- `--strict` - Enable strict validation mode
- `--format` - Output format (table, json, yaml)
- `--verbose` - Show detailed validation results
- `--help` - Show help for validate command

## Validation Checks

### Configuration Validation
- **Syntax**: YAML syntax correctness
- **Structure**: Required fields and sections
- **References**: File path references exist
- **Values**: Value ranges and formats

### Schema Validation
- **AVRO Schemas**: Schema syntax and compatibility
- **JSON Schemas**: Valid JSON Schema format
- **Field Types**: Supported data types
- **Schema Evolution**: Compatibility with existing data

### SQL Validation
- **Syntax**: FlinkSQL syntax correctness
- **References**: Table and column references
- **Functions**: Available function usage
- **Compatibility**: Flink version compatibility

### Connectivity Validation
- **Kafka**: Broker connectivity and topic access
- **Flink**: Cluster availability and resources
- **External Systems**: Database and API connectivity

## Validation Results

### Success Output
```
✅ Configuration validation passed
✅ Schema validation passed (3 schemas)
✅ SQL validation passed (4 files)
✅ Connectivity validation passed

All validations completed successfully!
```

### Error Output
```
❌ Configuration validation failed
  - Missing required field: pipeline.name
  - Invalid broker address: localhost:909

⚠️  Schema validation warnings
  - Schema compatibility issue in input.avsc
  - Deprecated field usage in output.avsc

❌ SQL validation failed
  - Syntax error in 01_create_source_table.sql:15
  - Unknown function 'CUSTOM_AGGREGATE' in processing.sql:23

2 errors, 2 warnings found
```

## Configuration Validation

Validates the main configuration file:

```yaml
# config.yaml
pipeline:
  name: "my-pipeline"  # ✅ Required field present
  version: "1.0.0"     # ✅ Valid version format

kafka:
  brokers:
    - "localhost:9092" # ✅ Valid broker format
  topic: "input"       # ✅ Topic name valid

flink:
  parallelism: 4       # ✅ Valid parallelism value
  memory: "2gb"        # ✅ Valid memory format
```

## Schema Validation

### AVRO Schema Example
```json
{
  "type": "record",
  "name": "UserEvent",
  "fields": [
    {
      "name": "userId",
      "type": "string"
    },
    {
      "name": "timestamp",
      "type": "long",
      "logicalType": "timestamp-millis"
    },
    {
      "name": "eventType",
      "type": "string"
    }
  ]
}
```

### JSON Schema Example
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "properties": {
    "userId": {
      "type": "string",
      "minLength": 1
    },
    "timestamp": {
      "type": "integer",
      "minimum": 0
    }
  },
  "required": ["userId", "timestamp"]
}
```

## SQL Validation

Validates FlinkSQL files for syntax and compatibility:

```sql
-- Valid FlinkSQL
CREATE TABLE source_table (
    user_id STRING,
    event_time TIMESTAMP(3),
    event_type STRING,
    WATERMARK FOR event_time AS event_time - INTERVAL '5' SECOND
) WITH (
    'connector' = 'kafka',
    'topic' = 'input-topic',
    'properties.bootstrap.servers' = 'localhost:9092',
    'format' = 'avro'
);
```

## Strict Mode

Enable stricter validation with `--strict`:

- **Deprecation Warnings**: Treat as errors
- **Best Practices**: Enforce coding standards
- **Performance**: Check for performance anti-patterns
- **Security**: Validate security configurations

## Integration with CI/CD

Use validate in your pipeline:

```yaml
# GitHub Actions example
- name: Validate Pipeline
  run: |
    pipegen validate --strict --format json > validation-results.json
    if [ $? -ne 0 ]; then
      echo "Validation failed"
      exit 1
    fi
```

## Exit Codes

- `0`: All validations passed
- `1`: Validation errors found
- `2`: Configuration file not found
- `3`: Invalid command line arguments

## Performance Tips

1. **Incremental Validation**: Only validate changed files
2. **Parallel Checks**: Run independent validations concurrently
3. **Caching**: Cache validation results for unchanged files
4. **Remote Validation**: Validate against live systems sparingly

## See Also

- [Configuration](../configuration.md)
- [pipegen run](./run.md)
- [pipegen check](./check.md)
