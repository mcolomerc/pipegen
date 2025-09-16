# Execution Report UX and Metrics Fixes

This issue tracks several improvements and fixes to the HTML Execution Report and the data used to render it.

Areas involved:
- Template: `internal/templates/files/execution_report.html`
- Dashboard/metrics types and server: `internal/dashboard/server.go`
- Execution data collection: `internal/dashboard/execution_data_collector.go`
- Kafka/Flink runtime metrics: `internal/pipeline/*.go`, `internal/docker/*.go`

## Problems to fix

1) Performance Metrics card should be a well-formatted table
- Current: A badge-like card shows aggregated metrics; it’s not aligned with other sections’ table layout and is hard to scan.
- Expected: Use the same table style as other cards (e.g., `.metrics-table`) with rows like: Total Messages Produced, Total Messages Consumed, Producer Rate (msgs/sec), Consumer Rate (msgs/sec), Bytes In/Out, Errors, Duration, etc.

2) Consumer Rate unit shows "0.00m sg/sec"
- Current: Unit string appears as `m sg/sec` and value is often 0.00.
- Expected: Correct unit to `msgs/sec` (messages per second). Compute value consistently with producer rate over the execution window.

3) Flink Jobs section lacks details/metrics
- Current: Only minimal information for Flink job(s).
- Expected: Add more job details and key metrics. Suggested fields: Job Name, Job ID, Status, Uptime, Parallelism, Number of Tasks, Throughput (records/sec), Backpressure, Checkpoint interval/duration/failure count, Last checkpoint time, Watermarks (min/max/avg), Task failures, Busy time, CPU/Mem if available.

4) Kafka topics metrics should be richer
- Current: Topics table has limited columns.
- Expected: Add per-topic columns such as: Partitions, Replication Factor, Log End Offset (per partition aggregated), Consumer Lag (per consumer group aggregated), Bytes In/Out (avg), Records In/Out (avg), Retention (hrs/size), Cleanup Policy (compact/delete), and optionally under-replicated status.

## Proposed approach

- Template: Extend `internal/templates/files/execution_report.html` to render the Performance Metrics card using the existing `.metrics-table` style for consistency with other cards.
- Units: Replace any literal `m sg/sec` or inconsistent unit strings with `msgs/sec`. Centralize unit formatting helpers if needed.
- Data model:
  - Extend `ProducerMetrics` and `ConsumerMetrics` in `internal/dashboard/server.go` with fields needed for the table (e.g., `TotalMessages`, `BytesIn`, `BytesOut`, `ErrorCount`, `DurationSec`, `RateMsgsPerSec`).
  - Extend `KafkaMetrics` to hold per-topic details: `TopicDetails []TopicDetail` with fields for partitions, replication, offsets, bytes/records rates, retention, cleanup policy, lag, and URP.
  - Add a `FlinkJobMetrics` struct with the suggested fields; include `[]FlinkJobMetrics` on the main payload.
- Collection:
  - Update `internal/dashboard/execution_data_collector.go` (and any related collectors) to capture and compute consumer rate, bytes/records rates, totals, and job/topic details.
  - Where not directly accessible, consider lightweight queries to Kafka (e.g., Admin API or `kafka-go`), and to Flink’s REST API (if already used elsewhere) to populate job metrics.
- Rendering:
  - Use `.metrics-table` for the Performance Metrics section with label/value rows for readability.
  - Expand Flink Job and Kafka Topics sections with the new fields as additional columns/rows.

## Acceptance criteria

- [ ] Performance Metrics section is rendered as a table using `.metrics-table`, matching the visual style of other cards.
- [ ] All rate units use `msgs/sec`; no occurrences of `m sg/sec` remain in the rendered report or template.
- [ ] Consumer Rate displays a non-zero value during active consumption for test scenarios used in CI/e2e; calculation matches produced/consumed counts over time windows.
- [ ] Flink Jobs section shows: Job Name, Job ID, Status, Uptime, Parallelism, #Tasks, Throughput (records/sec), Backpressure, Checkpoint stats (interval/duration/last/failed count), and (if available) CPU/Mem/Busy time.
- [ ] Kafka Topics table includes: Topic, Partitions, Replication Factor, Log End Offset (agg), Consumer Lag (agg), Bytes In/Out (avg), Records In/Out (avg), Retention, Cleanup Policy, URP flag.
- [ ] No layout regressions on desktop and mobile breakpoints; tables wrap gracefully and remain readable.
- [ ] New/updated fields are covered by unit tests where computation occurs, and by a lightweight HTML snapshot test for the template rendering.

## Notes

- Any new external calls (Flink REST, Kafka Admin) should be behind clear interfaces for testability and be disabled when not applicable in local report generation.
- Keep report generation resilient: missing metrics should display `N/A` gracefully without breaking layout.

## References
- Template: `internal/templates/files/execution_report.html`
- Types/Payload: `internal/dashboard/server.go`
- Collector: `internal/dashboard/execution_data_collector.go`
- Kafka/Flink runtime: `internal/pipeline/*.go`, `internal/docker/*.go`
