# Feature: Dataset-driven consumer validator (compare output to expected dataset)

## Summary
Complementing Issue #9 (dataset-driven producer), add a consumer validation mode that reads records from the output topic(s) and verifies them against a user-provided expected dataset (CSV/JSON/Parquet). This enables deterministic end-to-end checks for `pipegen run` scenarios using realistic input/output fixtures.

## Goals
- Allow the user to specify an expected output dataset file. After a run, the consumer validates that consumed records match the expected ones.
- Support CSV, JSON (NDJSON/array), and Parquet as expected dataset formats.
- Provide flexible matching strategies: exact match, unordered set equality, key-based alignment, and tolerance options for numeric/time fields.
- Produce a concise validation report with counts, mismatches, and sample diffs.

## Non-goals
- Full semantic diff/SQL correctness proof; this is a pragmatic content comparison.

## CLI UX
- New flags for `pipegen run` (or a subcommand `pipegen verify`):
  - `--expected-dataset <path>`: Path to expected output file.
  - `--expected-format <csv|json|parquet>` and `--expected-json-lines`.
  - `--expected-topic <name>`: Output topic to validate; defaults from project (e.g., `output-results`).
  - `--match-mode <exact|set|key>`: Default `set` (order-insensitive).
  - `--key-fields <k1,k2,...>`: For key mode, list of fields to join/compare.
  - `--numeric-tolerance <abs|rel:value>`: e.g., `abs:0.001` or `rel:0.01`.
  - `--time-tolerance <duration>`: e.g., `2s`.
  - `--max-wait <duration>`: How long to wait for expected count before asserting completeness.

## Implementation plan
1) Expected dataset reader
- Reuse or extend `internal/dataset/reader.go` from Issue #9 to parse the expected file into normalized rows with typed values.
- If an output AVRO schema exists, coerce expected rows to those types for fair comparison.

2) Consumer capture and decoding
- Extend `internal/pipeline/consumer.go` or add `validator_consumer.go` to:
  - Consume from the specified topic, decode AVRO (Schema Registry) or JSON.
  - Collect rows up to expected count (from expected dataset size) or until `--max-wait` timeout.

3) Matching and diff
- Implement a comparator:
  - exact: compare sequences 1:1
  - set: compare as unordered sets (multiset/bag semantics)
  - key: join on key fields and compare row-by-row
- Add tolerance handlers for numeric and time fields; pretty-print diffs (up to N examples per mismatch type).
- Emit a summary with totals: expected, actual, matched, missing, unexpected, field-level diffs.

4) Wiring and reports
- Add a `--report out.json` option to write a machine-readable result (pass/fail, statistics, samples).
- Integrate with `internal/dashboard` if helpful to surface pass/fail and counts.

5) Tests
- Unit tests for comparator modes (exact, set, key) including tolerance logic.
- Integration test: run a small pipeline with dataset producer + consumer validator to assert full pass.

6) Docs
- Add docs to `docs-site/run-workflow.md` and `docs-site/examples.md` on using expected datasets for validation.

## Acceptance criteria
- Given an expected dataset with N rows, running validation consumes N records from the topic (or times out) and produces a clear pass/fail result with a summary.
- Supports CSV, JSON (NDJSON/array), Parquet expected files.
- Works with AVRO-encoded topics (Schema Registry) and JSON fallbacks.
- Comparator modes and tolerances behave as documented.

## Risks and considerations
- Large outputs: streaming compare or sampled diffs to avoid memory blow-up.
- AVRO union types: null handling and type coercion need care for faithful comparisons.
- Partitioning and duplicates: set mode should treat duplicates with multiset counting, not pure set semantics.

## Proposed changes (by file)
- `internal/dataset/reader.go` (+ tests): reusable dataset readers for expected data.
- `internal/pipeline/validator_consumer.go` (or extend `consumer.go`): capture and compare output against expected.
- `cmd/run.go` or new `cmd/verify.go`: flags and execution flow for validation mode.
- `internal/dashboard/*`: optional pass/fail surfacing.
- `docs-site/*`: user docs and examples.
