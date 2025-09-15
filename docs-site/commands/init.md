# pipegen init

Create a new PipeGen project with SQL, AVRO schemas, config, and optional AI-powered generation.

## Usage

```bash
pipegen init <project-name> [flags]
```

## Flags

- `--force`           Overwrite existing project directory
- `--input-schema`    Path to an AVRO schema (input.avsc) to seed the project
- `--describe`        Natural language description for AI generation
- `--domain`          Business domain for better AI context (e.g., ecommerce, fintech, iot)
- `--help`            Show help

## Examples

```bash
# Basic project
pipegen init my-pipeline

# Initialize from an existing AVSC (schema-driven)
pipegen init payments --input-schema ./schemas/input.avsc

# AI generation from description
export PIPEGEN_OLLAMA_MODEL=llama3.1   # or set PIPEGEN_OPENAI_API_KEY
pipegen init fraud-detection --describe "Real-time fraud detection for transactions" --domain fintech

# Ground AI with your AVSC (combine flags)
pipegen init analytics \
  --input-schema ./schemas/input.avsc \
  --describe "Hourly revenue per user with watermarking" \
  --domain ecommerce
```

## Behavior

- Without `--describe` (standard generation)
  - Uses your `--input-schema` if provided, otherwise generates a default AVRO schema
  - Synthesizes a baseline Flink source DDL at `sql/01_create_source_table.sql` (Kafka + avro-confluent)

- With `--describe` (AI generation)
  - If AI is configured, PipeGen generates schemas, SQL, and docs; when `--input-schema` is provided, the AI is grounded on your schema and `schemas/input.avsc` is kept canonical
  - If AI is not configured, PipeGen falls back automatically:
    - With `--input-schema`: schema-driven generation with baseline DDL
    - Without `--input-schema`: minimal project with default schema and templates

## Generated Files

When you run `pipegen init`, it creates:

- `.pipegen.yaml` - Project configuration
- `schemas/` - AVRO schemas
  - `input.avsc` (canonical input schema)
  - `output_result.avsc` (AI path)
- `sql/` - Flink SQL files (includes `01_create_source_table.sql` in schema-driven path)
- `docker-compose.yml`, `flink-conf.yaml`, `flink-entrypoint.sh` (local stack)
- `README.md` - Project documentation
- `sql/OPTIMIZATIONS.md` - AI optimization suggestions (AI path)

## AI Providers

Configure one of the following to enable AI:

- Ollama (local): `PIPEGEN_OLLAMA_MODEL=llama3.1` and optionally `PIPEGEN_OLLAMA_URL=http://localhost:11434`
- OpenAI (cloud): `PIPEGEN_OPENAI_API_KEY=...` and optionally `PIPEGEN_LLM_MODEL=gpt-4o-mini`

If neither is set, `--describe` gracefully falls back as described above.

## Next Steps

1. `cd <project-name>`
2. Review generated `schemas/` and `sql/`
3. Start the local stack: `pipegen deploy`
4. Run the pipeline: `pipegen run`

See also: [AI Generation](../ai-generation.md), [Getting Started](../getting-started.md)
