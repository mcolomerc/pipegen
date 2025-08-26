# pipegen init

The `init` command creates a new PipeGen project with the necessary configuration files and templates.

## Usage

```bash
pipegen init [project-name] [flags]
```

## Examples

```bash
# Initialize a new project in current directory
pipegen init

# Initialize a new project with specific name
pipegen init my-pipeline

# Initialize with specific configuration
pipegen init my-pipeline --config cloud
```

## Flags

- `--config` - Configuration template to use (local, cloud)
- `--schema` - Schema type to initialize with (avro, json)
- `--help` - Show help for init command

## Generated Files

When you run `pipegen init`, it creates:

- `config.yaml` - Main configuration file
- `schemas/` - Directory for input/output schemas
- `sql/` - Directory for SQL processing files
- `README.md` - Project documentation

## Configuration Templates

### Local Configuration
- Sets up Kafka and Flink locally
- Uses Docker Compose
- Suitable for development and testing

### Cloud Configuration
- Configured for cloud deployment
- Production-ready settings
- Includes monitoring and scaling options

## Next Steps

After initializing your project:

1. Review and customize the generated `config.yaml`
2. Define your data schemas in the `schemas/` directory
3. Write your processing logic in the `sql/` directory
4. Run your pipeline with `pipegen run`

## See Also

- [Getting Started](../getting-started.md)
- [Configuration](../configuration.md)
- [pipegen run](./run.md)
