# Performance Optimization

Learn how to optimize your PipeGen pipelines for maximum throughput, minimum latency, and efficient resource utilization.

## Performance Fundamentals

### Key Metrics
- **Throughput**: Messages processed per second
- **Latency**: End-to-end processing time
- **Resource Utilization**: CPU, memory, network, and disk usage
- **Error Rate**: Percentage of failed message processing

### Performance Trade-offs
- **Throughput vs Latency**: Higher throughput often increases latency
- **Memory vs Processing Speed**: More memory can improve processing speed
- **Consistency vs Performance**: Stronger consistency guarantees impact performance

## Kafka Optimization

### Producer Configuration

```yaml
kafka:
  producer:
    # Increase batch size for higher throughput
    batch_size: 65536
    
    # Add small delay to allow batching
    linger_ms: 10
    
    # Use compression to reduce network usage
    compression_type: "lz4"
    
    # Optimize for throughput
    acks: "1"  # Use "all" for stronger consistency
    
    # Increase buffer memory for high-volume scenarios
    buffer_memory: 67108864  # 64MB
    
    # Optimize retries
    retries: 2147483647
    retry_backoff_ms: 100
```

### Consumer Configuration

```yaml
kafka:
  consumer:
    # Increase fetch size for better throughput
    fetch_min_bytes: 50000
    fetch_max_wait_ms: 500
    
    # Process more records per poll
    max_poll_records: 1000
    
    # Optimize memory usage
    receive_buffer_bytes: 65536
    send_buffer_bytes: 131072
    
    # Parallel processing
    max_poll_interval_ms: 300000
```

### Topic Configuration

```bash
# Create topic with optimal partitions
kafka-topics.sh --create \
  --topic high-throughput-topic \
  --partitions 12 \
  --replication-factor 3 \
  --config segment.ms=86400000 \
  --config retention.ms=259200000
```

## Flink Optimization

### Parallelism Configuration

```yaml
flink:
  # Set parallelism based on available cores and data volume
  parallelism: 8
  
  # Allow dynamic scaling
  max_parallelism: 128
  
  # Operator-specific parallelism in SQL
  # SELECT /*+ OPTIONS('sink.parallelism'='4') */ ...
```

### Memory Configuration

```yaml
flink:
  memory:
    # JobManager memory allocation
    jobmanager: "2gb"
    
    # TaskManager memory allocation
    taskmanager: "8gb"
    
    # Network buffers for high throughput
    network: "256mb"
    
    # Managed memory for state and caching
    managed:
      fraction: 0.4
      
  # JVM settings
  jvm_args:
    - "-XX:+UseG1GC"
    - "-XX:MaxGCPauseMillis=200"
    - "-XX:+UnlockExperimentalVMOptions"
    - "-XX:+UseCGroupMemoryLimitForHeap"
```

### Checkpointing Optimization

```yaml
flink:
  checkpoint:
    # Balance between recovery time and performance
    interval: "60s"  # Increase for higher throughput
    timeout: "10m"
    
    # Minimize checkpoint overhead
    min_pause: "10s"
    max_concurrent: 1
    
    # Use incremental checkpoints for large state
    incremental: true
    
    # Optimize storage
    storage: "s3://checkpoint-bucket/checkpoints"
    compression: true
```

### State Backend Optimization

```yaml
flink:
  state:
    backend: "rocksdb"  # Better for large state
    
    rocksdb:
      # Optimize RocksDB performance
      block_cache_size: "256mb"
      write_buffer_size: "128mb"
      max_write_buffer_number: 4
      
      # Compaction settings
      compaction_style: "level"
      max_background_jobs: 4
```

## SQL Optimization

### Query Optimization

```sql
-- Use appropriate time windows
SELECT 
    user_id,
    COUNT(*) as event_count,
    AVG(value) as avg_value
FROM source_table
WHERE event_time > NOW() - INTERVAL '1' HOUR
GROUP BY 
    user_id,
    TUMBLE(event_time, INTERVAL '5' MINUTE)
HAVING COUNT(*) > 10;  -- Filter early

-- Optimize joins with proper hints
SELECT /*+ USE_HASH_AGG, BROADCAST(dim) */
    fact.user_id,
    dim.user_name,
    SUM(fact.amount)
FROM fact_table fact
JOIN dimension_table /*+ OPTIONS('lookup.cache'='SYNC') */ dim
ON fact.user_id = dim.user_id;
```

### Index Usage

```sql
-- Create indexes for faster lookups
CREATE TABLE user_lookup (
    user_id STRING,
    user_name STRING,
    created_at TIMESTAMP(3),
    PRIMARY KEY (user_id)
) WITH (
    'connector' = 'jdbc',
    'lookup.cache' = 'SYNC',
    'lookup.max-retries' = '3'
);
```

## Resource Optimization

### CPU Optimization

#### TaskManager Configuration
```yaml
flink:
  taskmanager:
    # Number of task slots per TaskManager
    slots: 4
    
    # CPU allocation per slot
    cpu_cores: 2.0
    
    # Thread pool sizes
    rpc_threads: 4
    query_threads: 4
```

#### Processing Optimization
```sql
-- Use parallel processing hints
SELECT /*+ OPTIONS('sink.parallelism'='8') */
    user_id,
    SUM(amount)
FROM high_volume_stream
GROUP BY user_id;

-- Optimize windowing
SELECT 
    user_id,
    SUM(amount) as total
FROM events
GROUP BY 
    user_id,
    TUMBLE(event_time, INTERVAL '1' MINUTE)  -- Smaller windows = higher parallelism
```

### Memory Optimization

#### Memory Pool Configuration
```yaml
flink:
  memory:
    # Heap memory allocation
    heap:
      size: "6gb"
      
    # Off-heap memory for network buffers
    off_heap:
      size: "2gb"
      
    # Managed memory for RocksDB
    managed:
      size: "4gb"
      
    # Direct memory for serialization
    direct:
      size: "1gb"
```

#### Memory Usage Best Practices
- Use object reuse to reduce GC pressure
- Optimize serialization with custom serializers
- Monitor memory usage with metrics
- Use incremental checkpoints for large state

### Network Optimization

#### Network Buffer Configuration
```yaml
flink:
  network:
    # Number of network buffers
    buffers_per_channel: 8
    extra_buffers_per_gate: 8
    
    # Buffer size
    buffer_size: "32kb"
    
    # Buffer timeout for low latency
    buffer_timeout: "100ms"
```

#### Network Topology
- Co-locate related processing to reduce network overhead
- Use local storage for checkpoints when possible
- Configure proper network bandwidth allocation

## Performance Monitoring

### Key Performance Indicators

```yaml
monitoring:
  metrics:
    # Throughput metrics
    - "flink_taskmanager_job_task_operator_numRecordsInPerSecond"
    - "flink_taskmanager_job_task_operator_numRecordsOutPerSecond"
    
    # Latency metrics
    - "flink_taskmanager_job_latency_source_id_operator_id_operator_subtask_index_latency"
    
    # Resource metrics
    - "flink_memory_heap_used"
    - "flink_cpu_usage"
    
    # Checkpoint metrics
    - "flink_jobmanager_job_lastCheckpointDuration"
    - "flink_jobmanager_job_lastCheckpointSize"
```

### Performance Dashboard

Create Grafana dashboards to monitor:

```json
{
  "dashboard": {
    "title": "PipeGen Performance",
    "panels": [
      {
        "title": "Throughput (msg/s)",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(flink_taskmanager_job_task_operator_numRecordsInPerSecond[5m])"
          }
        ]
      },
      {
        "title": "Processing Latency",
        "type": "graph",
        "targets": [
          {
            "expr": "flink_taskmanager_job_latency_source_id_operator_id_operator_subtask_index_latency"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "flink_memory_heap_used / flink_memory_heap_max * 100"
          }
        ]
      }
    ]
  }
}
```

## Performance Testing

### Load Testing Strategy

```bash
# Generate test load
pipegen run \
  --messages 1000000 \
  --rate 10000 \
  --traffic-pattern "0:1000,300:10000,600:5000,900:15000" \
  --config performance-test.yaml

# Monitor during test
pipegen check --watch --interval 10s
```

### Benchmarking

```yaml
# benchmark.yaml
benchmark:
  scenarios:
    - name: "high_throughput"
      messages: 10000000
      rate: 50000
      duration: "10m"
      
    - name: "low_latency"
      messages: 1000000
      rate: 1000
      latency_target: "10ms"
      
    - name: "mixed_load"
      traffic_pattern: "0:1000,120:10000,240:500,360:20000"
      duration: "20m"
```

## Scaling Strategies

### Horizontal Scaling

```yaml
# Scale TaskManagers
flink:
  taskmanager:
    replicas: 8
    slots: 4
    
# Scale Kafka partitions
kafka:
  topics:
    input:
      partitions: 16
      replication_factor: 3
```

### Vertical Scaling

```yaml
# Increase resources per node
flink:
  memory:
    taskmanager: "16gb"
  cpu:
    taskmanager: 8
```

### Auto-scaling

```yaml
# Kubernetes HPA configuration
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: flink-taskmanager-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: flink-taskmanager
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Troubleshooting Performance Issues

### Common Performance Problems

#### High Latency
```bash
# Check processing backlog
pipegen dashboard --metrics latency

# Analyze checkpoint duration
curl http://jobmanager:8081/jobs/overview | jq '.jobs[0].duration'

# Monitor network buffers
grep "network buffer" flink-taskmanager.log
```

#### Low Throughput
```bash
# Check parallelism utilization
pipegen check --verbose | grep -i parallel

# Monitor Kafka consumer lag
kafka-consumer-groups.sh --bootstrap-server kafka:9092 \
  --describe --group pipegen-consumer

# Check resource utilization
docker stats flink-taskmanager
```

#### Memory Issues
```bash
# Check heap usage
jstat -gc $(pgrep -f taskmanager) 5s

# Monitor off-heap memory
grep "Direct memory" flink-taskmanager.log

# Check for memory leaks
jmap -histo $(pgrep -f taskmanager) | head -20
```

### Performance Tuning Checklist

- [ ] Set appropriate parallelism levels
- [ ] Configure memory pools optimally
- [ ] Tune Kafka producer/consumer settings
- [ ] Optimize SQL queries and operators
- [ ] Enable incremental checkpoints
- [ ] Monitor key performance metrics
- [ ] Test with realistic load patterns
- [ ] Plan for horizontal scaling
- [ ] Implement auto-scaling policies
- [ ] Regular performance reviews

## See Also

- [Configuration](../configuration.md)
- [Dashboard](../dashboard.md)
- [Troubleshooting](./troubleshooting.md)
