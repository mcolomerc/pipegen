# Feature: Improve generation from provided AVSC (schema-grounded AI SQL)

## Summary
Enhance `pipegen init` when the user supplies `--input-schema <path.avsc>` so Pipegen can:
- Treat the provided AVSC as the canonical input schema without modifications.
- Optionally call AI with the schema context (and `--describe` text) to generate SQL transformations and a pipeline description grounded in the schema.
- If AI is not available or `--describe` is not provided, still generate a high-quality Flink SQL `CREATE TABLE` for the input source that reflects the AVSC fields and logical types.

This enables schema-first workflows and better SQL generation aligned to the actual data model.

## Goals
- Allow combining `--input-schema` with `--describe` (lift current restriction).
- Ground AI generation in the provided schema for safer, more accurate SQL and documentation.
- Always produce a correct `01_create_source_table.sql` from AVSC even without AI.

## Non-goals
- Output schema derivation beyond what AI returns (if used). If no AI, we do not auto-generate an output schema.
- Inferring schemas from non-AVSC inputs (covered by the dataset feature).

## CLI UX
- Update `pipegen init` behavior:
  - Permit `--input-schema` + `--describe` together. If AI is enabled, the LLM is called with a prompt including the exact schema.
  - If `--describe` is omitted or AI disabled/unavailable, proceed with standard project scaffolding plus a generated source `CREATE TABLE` based on the AVSC fields.
- Optional new flags:
  - `--timestamp-col <name>` and `--watermark <duration>` to augment the DDL (WATERMARK clause). These are optional; if absent, no watermark unless detected by AVSC doc annotations (future work).

### Examples
- `pipegen init orders --input-schema ./schemas/orders.avsc --describe "aggregate revenue by product per hour" --domain ecommerce`
- `pipegen init logs --input-schema ./event.avsc --timestamp-col event_time --watermark 5m`

## Implementation plan
1) Lift CLI restriction
- In `cmd/init.go`, remove the error that prevents `--input-schema` with `--describe`.
- If both are provided and AI is enabled, call AI with schema-grounding.

2) Schema-grounded AI prompt
- Extend `internal/llm/service.go` with either:
  - New method `GenerateFromSchema(ctx, schemaJSON, description, domain)`; or
  - Augment `buildPrompt()` to include a section:
    - `Input schema (AVRO JSON): <pretty JSON>`
- Keep existing JSON-only response contract. Reuse `parseResponse()`.

3) Generator support for CREATE TABLE from AVSC
- Add a helper in `internal/generator` to synthesize a Flink SQL DDL from the AVSC fields, including:
  - Column list with proper types, handling logical types (date, timestamp-millis/micros, decimal when feasible)
  - Optional `WATERMARK FOR <ts> AS <ts> - INTERVAL '<watermark>'`
  - Kafka or filesystem connector config based on local mode (default to Kafka as today, or filesystem when `--filesystem-source` flag is set—TBD)
- Write the DDL to `sql/01_create_source_table.sql` when AI is not used, or when we want a baseline source DDL alongside AI output.

4) Plumb schema content
- When `--input-schema` provided, read the file content and pass to generator via a new method `SetInputSchemaContent(content string)` in addition to the existing `SetInputSchemaPath()` (or replace uses accordingly).
- Ensure the file is copied to `schemas/input.avsc` as today for consistency.

5) Tests
- Add unit tests that:
  - Verify the DDL column mapping for a variety of AVRO types, including logical types.
  - Validate that combining `--input-schema` + `--describe` triggers the AI call when enabled and falls back gracefully when disabled.

6) Docs
- Update `docs-site/commands/init.md` to note that `--input-schema` can be combined with `--describe`, and document the generated DDL behavior and optional watermark flags.

## Acceptance criteria
- Running `pipegen init proj --input-schema file.avsc` creates:
  - `schemas/input.avsc` copied from user file
  - `sql/01_create_source_table.sql` reflecting the AVSC fields and types; includes WATERMARK when flags provided
- Running with `--describe` and AI enabled produces:
  - AI-generated SQL and description grounded by the provided schema, plus `OPTIMIZATIONS.md` when present
- Existing behavior remains unchanged when neither `--input-schema` nor `--describe` are provided.

## Risks and considerations
- Mapping AVRO unions and complex types to Table SQL types—initially support common primitives and logical types; for unions with null, unwrap the non-null type.
- Decimal handling relies on schema having `logicalType`, `precision`, and `scale`—if missing, fallback to DOUBLE or BYTES with a note.

## Proposed changes (by file)
- `cmd/init.go`: allow `--input-schema` + `--describe`; read schema content for AI prompt; wire new flags (optional).
- `internal/llm/service.go`: add schema-grounded generation path.
- `internal/generator/generator.go`: add AVSC->DDL utility and `SetInputSchemaContent`.
- `docs-site/commands/init.md`: document combined usage and examples.
