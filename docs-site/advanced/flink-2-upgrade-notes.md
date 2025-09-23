# Flink 2.x Upgrade Notes (Work In Progress)

Tracking issue: https://github.com/mcolomerc/pipegen/issues/15

## Target Versions
- Flink Runtime / Image: `flink:2.1.0-scala_2.12` (verify latest stable before final merge)
- Kafka Connectors (DataStream & SQL): from 3.x line (1.x compatibility) → use `4.0.1-2.0` for Flink 2.x
- Avro Confluent Registry: bump from `1.18.1` → `2.1.0`
- Format Libraries: `flink-json:2.1.0`, `flink-csv:2.1.0`

## Current (Pre-Upgrade) Connectors
```
flink-connector-kafka-3.1.0-1.18.jar
flink-sql-connector-kafka-3.1.0-1.18.jar
flink-sql-avro-confluent-registry-1.18.1.jar
kafka-clients-3.4.0.jar
jackson-core-2.15.2.jar
jackson-databind-2.15.2.jar
jackson-annotations-2.15.2.jar
guava-31.1-jre.jar
flink-json-1.18.1.jar
flink-csv-1.18.1.jar
```

## Proposed (Flink 2.x) Connectors
```
flink-connector-kafka-4.0.1-2.0.jar
flink-sql-connector-kafka-4.0.1-2.0.jar
flink-sql-avro-confluent-registry-2.1.0.jar
kafka-clients-3.7.0.jar  # verify minimum required
jackson-core-2.17.2.jar
jackson-databind-2.17.2.jar
jackson-annotations-2.17.2.jar
guava-33.2.1-jre.jar
flink-json-2.1.0.jar
flink-csv-2.1.0.jar
```
(Exact versions subject to validation of transitive compatibility; we may allow minor drift if Flink image already bundles newer libs.)

## Action Items
- [ ] Update `internal/templates/files/docker/connectors.txt`
- [ ] Adjust download logic if it assumes `-1.18` suffix patterns
- [ ] Review license / NOTICE implications for updated artifacts
- [ ] Smoke test: pipeline init + deploy + run with Kafka topic
- [ ] Validate Avro Confluent schema evolution still works
- [ ] Update docs referencing old jar names (if any)

## Compatibility Considerations
| Area | Change | Action |
|------|--------|--------|
| Connector artifact naming | Suffix moves from `-1.18` to `-2.0` | Replace patterns & tests |
| Avro registry jar | Major bump | Verify serialization & schema fetch |
| Jackson libs | Upstream security/bugfix updates | Keep aligned versions to avoid classpath conflicts |
| Kafka clients | 3.4.0 → 3.7.x | Confirm broker compatibility window |

## Testing Strategy
1. Generate project (no AI) → verify jars downloaded with new names
2. Deploy stack → confirm Flink JobManager starts without missing classes
3. Submit SQL (source -> simple projection -> sink) → success
4. Run producer/consumer pattern with CSV ingestion
5. Dashboard metrics load (ensure REST fields unchanged or adapted)

## Open Questions
- Is scala_2.12 still the recommended tag for Flink 2.1.x or should we move to 2.13? (Investigate.)
- Do we supply both SQL & DataStream connectors or consolidate?

## Rollback Plan
If critical blocker discovered, revert connector list & image tag in a single revert commit and publish note in changelog.

---
(Initial draft; will evolve through PR review.)
