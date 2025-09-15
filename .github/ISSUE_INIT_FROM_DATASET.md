# Feature: Init project from dataset (CSV/JSON/Parquet) with schema inference and AI-assisted SQL

## Summary
Add a new initialization path to `pipegen init` that bootstraps a project directly from a dataset file (CSV, JSON, or Parquet). Pipegen will:
- Infer the input schema (to AVRO), including nullable types and logical types.
- Detect event-time columns and propose a WATERMARK strategy.
- Generate a source table (Flink SQL DDL) targeting a filesytem connector for local runs; optionally scaffold a Kafka source with the same schema.
- Create initial processing SQL (projection/window template) and a short, AI-authored pipeline description grounded on the inferred schema. If `--describe` is provided, the AI will generate richer SQL and optimizations based on the schema.

This streamlines POCs and lets users start from real data rather than hand-writing schemas.

## Goals
- One command to create a runnable project from a dataset path.
- Accurate schema inference for CSV/JSON/Parquet with sensible defaults.
- Event-time detection and watermark suggestion.
- Optional AI assistance grounded on the inferred schema.
- Keep current init flows working (default and `--input-schema`).

## Non-goals
- Full-blown data profiling beyond light sampling.
- Auto-detection of external systems (e.g., inferring Kafka cluster). We scaffold templates for local first.

## CLI UX
- New flags for `pipegen init`:
  - `--dataset <path>`: Path to a local dataset file (csv, json, parquet). Format auto-detected by extension; override with `--format`.
  - `--format <csv|json|parquet>`: Optional explicit format.
  - `--json-lines` (bool): For JSON that is line-delimited (NDJSON). Default: auto-detect; flag forces behavior.
  - `--timestamp-col <name>`: Explicit event-time column if detection disagrees.
  - `--timestamp-unit <s|ms|us|ns>`: Unit for epoch timestamps. Default: auto from data.
  - `--watermark <duration>`: Default `5m`. Example: `2m`, `30s`.
  - `--describe <text>` and `--domain <text>` continue to work and may be combined with `--dataset`.

Notes:
- Existing conflict rule in `cmd/init.go` prevents `--input-schema` with `--describe`. For `--dataset`, we allow using `--describe` (AI is grounded by the inferred schema).

### Examples
- `pipegen init orders-poc --dataset ./samples/orders.csv`
- `pipegen init web-logs --dataset ./logs.json --json-lines --timestamp-col ts --watermark 2m --describe "sessionize and top pages" --domain ecommerce`
- `pipegen init metrics --dataset ./metrics.parquet --format parquet`

## Implementation plan
1) Dataset inspector and schema inference
- New package: `internal/dataset` with:
  - `Detector`: picks format by extension/content sniffing.
  - `Sampler`: reads a bounded sample efficiently (e.g., first N KB/rows).
  - `Inferer`: produces a normalized field list (name, type, nullable, logicalType) with heuristics:
    - CSV: use `encoding/csv`, header detection, type inference across rows (int/double/bool/date/time/timestamp/string).
    - JSON: support NDJSON and array JSON; stream with `json.Decoder`; infer nested objects/arrays (flatten only 1 level initially, mark complex types as string or map with TODO config).
    - Parquet: use a pure-Go lib (e.g., `github.com/parquet-go/parquet-go`); map Parquet types/logical types to AVRO equivalents.
  - `EventTimeDetector`: heuristics on names (`event_time,timestamp,ts,created_at`) and sample values (ISO-8601 vs epoch). Outputs: column name, logical type (timestamp-millis|micros), suggested watermark duration.
- Produce AVRO schema JSON string with rules:
  - All fields default nullable: `{"type": ["null", T], "default": null}`.
  - Map numeric widths; decimals as `bytes` with logicalType decimal (defer if scale not clear).
  - Timestamps prefer logicalType `timestamp-millis`/`timestamp-micros`; dates as `int` with logicalType `date`.

2) Generator wiring
- Extend `internal/generator` to accept schema content:
  - Add `SetInputSchemaContent(json string)` on `ProjectGenerator` and branch in `generateAVROSchemas()` to write that content as `schemas/input.avsc`.
- Generate SQL DDL for source table(s):
  - For local mode: a `filesystem` connector DDL under `sql/local/01_create_source_table.sql` with proper `format` and options:
    - CSV: `'format'='csv', 'csv.ignore-parse-errors'='true', 'path'='<relative path>'`.
    - JSON: `'format'='json', 'json.ignore-parse-errors'='true'` and `'json.timestamp-format.standard'='ISO-8601'` when applicable.
    - Parquet: `'format'='parquet'`.
  - Insert `WATERMARK FOR <event_time> AS <event_time> - INTERVAL '<watermark>'` when detected or when `--timestamp-col` is provided.
  - Optionally scaffold a Kafka source DDL disabled by default (commented) with same schema.
- Create initial processing SQL:
  - `02_create_output_table.sql` with a minimal sink (filesystem or Kafka) and upsert semantics placeholder.
  - `03_create_processing.sql` with a projection or a windowed aggregation template if event-time exists (tumbling 1m window by default).

3) AI grounding
- Extend LLM prompt to include the inferred schema summary and optional sample fields when `--dataset` is used.
  - Option A: new method `GenerateFromSchema(ctx, schemaJSON, description, domain)`.
  - Option B: augment `buildPrompt()` to include a section `Input schema:` when available.
- Keep `PIPEGEN_MOCK_OPENAI` path for tests.

4) CLI changes (`cmd/init.go`)
- Add flags listed above. Permit `--dataset` + `--describe`.
- Flow:
  - If `--dataset` present: run inspector -> get `schemaJSON`, `eventTime`, `watermark`, and DDL snippets.
  - If `--describe`: call LLM with schema context to generate AI SQL + description.
  - Write project with generator using `SetInputSchemaContent(schemaJSON)` and generated SQL files (for AI case, route through `LLMProjectGenerator`).

5) Templates
- Add/extend SQL templates for filesystem sources in `internal/templates/files/sql/local`.
- Keep existing Docker/Flink templates; no change required to run with filesystem source.

6) Tests
- Unit tests under `internal/dataset` with small `testdata/`:
  - CSV with header and mixed types; JSON NDJSON and array; Parquet schema with timestamp.
  - Validate inferred AVRO and event-time detection.
- E2E test for CLI: `pipegen init myproj --dataset <file>` with `PIPEGEN_MOCK_OPENAI=true` verifying project shape and key file contents.

7) Docs
- Update `docs-site/commands/init.md` and `docs-site/examples.md` with the new flags and examples.
- Add a short section on limitations and overrides.

## Acceptance criteria
- Running `pipegen init myproj --dataset <csv/json/parquet>` creates a project with:
  - `schemas/input.avsc` matching inferred fields.
  - `sql/01_create_source_table.sql` with correct connector, schema, and watermark when applicable.
  - `sql/03_create_processing.sql` present; if `--describe` provided, AI-generated SQL appears and `OPTIMIZATIONS.md` is created.
  - `pipegen validate` passes on the scaffolded project.
- `--timestamp-col` overrides detection; `--watermark` overrides default.
- Works on Linux/macOS without CGO or heavyweight deps.

## Risks and considerations
- Large files: sampling only a bounded prefix to avoid heavy I/O.
- JSON complexity: nested structures may be flattened conservatively; deeper nesting deferred.
- Parquet libs: prefer `github.com/parquet-go/parquet-go` for pure Go. If we need another, gate behind build tags or optional module.
- Timezones and formats: keep defaults sane; expose overrides via flags.

## Proposed changes (by file)
- `cmd/init.go`: new flags, validation, branch for dataset flow, allow `--dataset` + `--describe`.
- `internal/dataset/*`: new detector/sampler/inferer/time detector implementation + tests.
- `internal/generator/generator.go`: support `SetInputSchemaContent` and conditionally write provided schema; helper to generate filesystem source DDL.
- `internal/llm/service.go`: accept optional schema context in prompts or new method.
- `internal/templates/files/sql/local/*`: add filesystem source template variants.
- `docs-site/commands/init.md`: document flags and examples.
- `cmd/validate_test.go` or new CLI E2E test: cover dataset init path.

## Open questions
- Do we want to create both filesystem and Kafka sources by default, or only filesystem and comment Kafka as a follow-up?
- Should we emit both `input.avsc` and `input_event.avsc` for backwards compatibility, or consolidate on one name?
- Minimum sampling size and performance budget for very large files?
