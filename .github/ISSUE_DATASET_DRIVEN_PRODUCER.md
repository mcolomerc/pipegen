# Feature: Dataset-driven producer (replay dataset rows with inferred schema)

## Summary
After implementing Issue #7 (init from dataset with schema inference), add a producer mode that publishes events sourced directly from the dataset contents rather than synthetic data. This enables realistic local testing and performance simulations, replaying real rows at a configured rate while honoring the inferred (or provided) schema.

## Goals
- Produce Kafka messages using rows from a provided dataset (CSV/JSON/Parquet) or the dataset referenced by the generated project.
- Use the inferred AVRO schema (from issue #7) or provided AVSC/JSON Schema to serialize messages (Schema Registry compatible).
- Support constant rate and traffic patterns; optionally replay timestamps from the dataset.
- If mixing with the existing synthetic producer is complex, implement a new producer component for dataset replay.

## Non-goals
- Full batch backfill tooling (covered separately by replay/backfill features).
- Complex join/transform before publish (out of scope; this producer only replays rows to the source topic).

## CLI UX
- New flags under `pipegen run` or a dedicated subcommand (TBD):
  - `--dataset <path>`: Path to data file (csv,json,parquet). If omitted, use the project’s dataset path recorded at init (to be stored in `.pipegen.yaml`).
  - `--format <csv|json|parquet>` and `--json-lines` for JSON when needed.
  - `--rate <msg/sec>` or traffic patterns via existing config.
  - `--loop` to cycle the dataset when reaching EOF.
  - `--timestamp-col <name>` and `--use-event-time` to send timestamps from the dataset (affects partitioning keys optional; mainly for observability alignment).
  - `--max-messages <n>` to cap total messages for test runs.

## Implementation plan
1) Data reader/iterator
- Add `internal/dataset/reader.go` implementing streaming readers for CSV (encoding/csv), JSON (NDJSON and array with json.Decoder), and Parquet (pure Go library e.g. parquet-go). Return rows as `map[string]interface{}` with typed fields.
- Normalize field names to match inferred schema; support type coercion (string->int/double/bool/date/timestamp) with best-effort conversions and warnings.

2) Schema application and encoding
- Reuse existing `pipeline.Schema` and `Producer.InitializeSchemaRegistry` to get codec and schema ID.
- Add a mapper to convert row maps into AVRO-native payloads using schema fields (handle unions with null by picking non-null type when value present; otherwise null).

3) Producer mode
- Extend `internal/pipeline/producer.go` or create `dataset_producer.go`:
  - Accept a `RowSource` interface with `Next() (map[string]interface{}, bool, error)`.
  - Drive emission at configured rate or via traffic patterns, similar to current code.
  - On EOF: if `--loop`, rewind; else stop gracefully.
  - Track metrics: rows read, messages sent, lag vs configured rate.

4) Config and wiring
- Extend `.pipegen.yaml` to record dataset path/format from init (#7) so `pipegen run` can default to it.
- Update `cmd/run.go` to recognize dataset mode flags and construct the `RowSource` accordingly.

5) Tests
- Unit tests for readers (CSV/JSON/Parquet) with `testdata/` fixtures.
- Integration test with MiniKafka (or mock writer) verifying AVRO serialization matches schema and rate control logic.

6) Docs
- Update `docs-site/run-workflow.md` and examples to show dataset-driven production, including rate control and looping behavior.

## Acceptance criteria
- Running dataset mode publishes messages to the configured source topic with AVRO encoding matching the project’s input schema.
- Works for CSV, JSON (NDJSON/array), and Parquet.
- Honors `--rate`/traffic patterns and `--loop`.
- Graceful stop at EOF when not looping; clear progress logs and metrics surfaced in Pipegen dashboard.

## Risks and considerations
- Parquet dependency footprint: prefer pure-Go; if not feasible, guard behind build tags or optional module.
- Type coercion errors: log warnings and count dropped rows; support `--ignore-parse-errors` behavior similar to Flink connectors.
- Large files: consider sampling or chunked reading to limit memory.

## Proposed changes (by file)
- `internal/dataset/reader.go` (+ tests): dataset row readers and normalization.
- `internal/pipeline/dataset_producer.go` (or extend `producer.go`): producer that emits dataset rows.
- `cmd/run.go`: flags and wiring for dataset mode.
- `.pipegen.yaml`: store dataset path/format from init (added in #7). 
- `docs-site/*`: usage docs and examples.
