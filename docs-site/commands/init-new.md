# pipegen init

Initialize a new PipeGen streaming pipeline project with comprehensive scaffolding and optional AI-powered generation.

## Usage

```bash
pipegen init <project-name> [flags]
```

## Examples

<div class="info-box">
  <i class="fas fa-terminal icon"></i>
  <div>
    <strong>Basic project initialization:</strong>
    <pre><code>pipegen init my-pipeline</code></pre>
  </div>
</div>

<div class="info-box">
  <i class="fas fa-robot icon"></i>
  <div>
    <strong>AI-powered project generation:</strong>
    <pre><code>pipegen init fraud-detection --describe "Real-time fraud detection for payment transactions using machine learning"</code></pre>
  </div>
</div>

<div class="info-box">
  <i class="fas fa-cloud icon"></i>
  <div>
    <strong>Cloud-optimized setup:</strong>
    <pre><code>pipegen init analytics-pipeline --cloud --provider aws</code></pre>
  </div>
</div>

## Flags

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--describe` | string | Natural language description for AI generation | - |
| `--cloud` | bool | Generate cloud-optimized configuration | false |
| `--provider` | string | Cloud provider (aws, gcp, azure) | aws |
| `--template` | string | Use predefined template | standard |
| `--force` | bool | Overwrite existing directory | false |
| `--verbose` | bool | Enable verbose output | false |

## Project Structure

When you run `pipegen init`, it creates a complete project structure:

```
my-pipeline/
├── .pipegen.yaml           # Project configuration
├── docker-compose.yml      # Local development stack
├── README.md               # Project documentation
├── config/
│   ├── local.yaml         # Local environment config
│   ├── cloud.yaml         # Cloud environment config
│   └── flink-conf.yaml    # Flink configuration
├── schemas/
│   ├── input.json         # Input AVRO schema
│   └── output.json        # Output AVRO schema
└── sql/
    ├── 01_create_source_table.sql
    ├── 02_create_processing.sql
    ├── 03_create_output_table.sql
    └── 04_insert_results.sql
```

## AI-Powered Generation

<div class="info-box">
  <i class="fas fa-magic icon"></i>
  <div>
    <strong>Natural Language Processing</strong><br>
    Describe your pipeline in plain English and let AI generate optimized SQL, schemas, and configuration.
  </div>
</div>

### Example Descriptions

```bash
# E-commerce analytics
pipegen init ecommerce --describe "Process customer orders, calculate daily revenue, detect trending products, and generate real-time sales dashboards"

# IoT monitoring  
pipegen init iot-monitoring --describe "Collect sensor data from industrial equipment, detect anomalies, predict maintenance needs, and alert on critical issues"

# Log processing
pipegen init log-analysis --describe "Parse application logs, extract error patterns, generate metrics, and create alerts for system health monitoring"
```

## Templates

Choose from predefined templates for common use cases:

| Template | Description | Use Case |
|----------|-------------|----------|
| `analytics` | Real-time analytics pipeline | Business metrics, KPIs |
| `fraud` | Fraud detection system | Payment processing, security |
| `iot` | IoT data processing | Sensor data, monitoring |
| `logs` | Log processing pipeline | Application monitoring |
| `social` | Social media analytics | Content analysis, trends |

```bash
pipegen init my-project --template fraud
```

## Next Steps

After initializing your project:

1. **<i class="fas fa-eye fa-icon"></i>Review generated files** - Check SQL, schemas, and configuration
2. **<i class="fas fa-edit fa-icon"></i>Customize as needed** - Modify generated code for your requirements  
3. **<i class="fas fa-rocket fa-icon"></i>Deploy development stack** - Run `pipegen deploy`
4. **<i class="fas fa-play fa-icon"></i>Start processing** - Execute `pipegen run`
5. **<i class="fas fa-tachometer-alt fa-icon"></i>Monitor pipeline** - Open dashboard with `pipegen dashboard`

## See Also

- [<i class="fas fa-play fa-icon"></i>pipegen run](./run.md) - Execute your pipeline
- [<i class="fas fa-check fa-icon"></i>pipegen validate](./validate.md) - Validate project structure
- [<i class="fas fa-cog fa-icon"></i>Configuration](../configuration.md) - Configuration reference
- [<i class="fas fa-robot fa-icon"></i>AI Generation](../ai-generation.md) - AI-powered features
