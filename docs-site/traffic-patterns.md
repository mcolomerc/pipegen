# Traffic Patterns

The `--traffic-pattern` feature allows you to simulate realistic traffic spikes during pipeline execution, perfect for load testing, capacity planning, and performance validation.

## Overview

Instead of sending messages at a constant rate, traffic patterns let you define **dynamic rate changes** during execution. This enables realistic simulation of:

- **Black Friday traffic spikes** 🛍️
- **Breaking news events** 📺  
- **System load variations** 📈
- **User behavior patterns** 👥

## Pattern Format

```bash
--traffic-pattern "start-end:rate%,start-end:rate%"
```

### Components

- **`start-end`**: Time range using flexible formats
  - `30s-60s` (30 seconds to 60 seconds)
  - `1m-2m` (1 minute to 2 minutes)  
  - `1m30s-2m45s` (complex timing)
- **`rate%`**: Message rate multiplier
  - `300%` = 3x base rate
  - `150%` = 1.5x base rate
  - `500%` = 5x base rate

## Basic Examples

### Single Traffic Spike

```bash
# 3x rate spike from 30s to 60s
pipegen run --message-rate 100 --duration 5m \
  --traffic-pattern "30s-60s:300%"
```

**Timeline:**
- `0-30s`: 100 msg/sec (base rate)
- `30s-60s`: **300 msg/sec** (3x spike)
- `60s-5m`: 100 msg/sec (back to base)

### Multiple Spikes

```bash
# Multiple traffic bursts
pipegen run --message-rate 50 --duration 10m \
  --traffic-pattern "1m-2m:400%,5m-6m:300%,8m-9m:500%"
```

**Timeline:**
- `0-1m`: 50 msg/sec
- `1m-2m`: **200 msg/sec** (4x)
- `2m-5m`: 50 msg/sec  
- `5m-6m`: **150 msg/sec** (3x)
- `6m-8m`: 50 msg/sec
- `8m-9m`: **250 msg/sec** (5x)
- `9m-10m`: 50 msg/sec

## Real-World Scenarios

### 🛍️ Black Friday Simulation

```bash
# Simulate e-commerce traffic during Black Friday
pipegen run --message-rate 100 --duration 20m \
  --traffic-pattern "5m-7m:300%,10m-15m:500%,17m-19m:400%"
```

Models typical shopping patterns:
- Early morning spike (5-7 minutes)
- Peak shopping hours (10-15 minutes) 
- Last-minute rush (17-19 minutes)

### 📺 Breaking News Event

```bash
# News website traffic during breaking news
pipegen run --message-rate 50 --duration 15m \
  --traffic-pattern "2m-5m:800%,8m-10m:400%"
```

Simulates:
- Massive initial spike when news breaks
- Sustained elevated traffic as story develops

### 🏢 Business Hours Pattern

```bash
# Gradual increase during business hours
pipegen run --message-rate 20 --duration 12m \
  --traffic-pattern "1m-2m:150%,3m-4m:200%,5m-6m:300%,7m-8m:250%,9m-10m:200%,11m-12m:150%"
```

### 🎮 Gaming Event Launch

```bash
# Gaming server load during event launch
pipegen run --message-rate 75 --duration 8m \
  --traffic-pattern "1m-3m:600%,5m-7m:400%"
```

## Advanced Features

### Validation & Safety

PipeGen automatically validates traffic patterns:

```bash
# ❌ This will fail - patterns overlap
pipegen run --traffic-pattern "30s-60s:300%,45s-90s:200%"

# ❌ This will fail - pattern exceeds duration
pipegen run --duration 2m --traffic-pattern "30s-180s:300%"

# ✅ This is valid
pipegen run --duration 5m --traffic-pattern "30s-60s:300%,90s-120s:200%"
```

### Dry Run Preview

Always preview your traffic pattern before execution:

```bash
pipegen run --message-rate 100 --duration 5m \
  --traffic-pattern "30s-60s:300%,90s-120s:200%" \
  --dry-run
```

**Output:**
```
📋 Execution Plan:
  Project Directory: .
  Traffic Pattern:
    Base rate: 100 msg/sec
      Peak 1: 30s-1m0s at 300 msg/sec (300%)
      Peak 2: 1m30s-2m0s at 200 msg/sec (200%)
  Duration: 5m0s
  Steps that would be executed:
  1. Load SQL statements from sql/ directory
  2. Load AVRO schemas from schemas/ directory
  ...
  7. Start Kafka producer with dynamic traffic patterns
  ...
```

## Performance Considerations

### Rate Calculation

- **Smooth transitions**: Rate changes are applied immediately but smoothly
- **Resource usage**: Higher rates consume more CPU and memory
- **Network impact**: Consider bandwidth limitations
- **Kafka capacity**: Ensure brokers can handle peak throughput

### Best Practices

1. **Start small**: Begin with modest multipliers (200-300%)
2. **Monitor resources**: Watch CPU, memory, and network usage
3. **Gradual increases**: Use stepping patterns rather than extreme jumps
4. **Test duration**: Allow time for pattern effects to be observable

### Example: Gradual Load Increase

```bash
# Gentle load increase pattern
pipegen run --message-rate 50 --duration 10m \
  --traffic-pattern "1m-2m:150%,3m-4m:200%,5m-6m:250%,7m-8m:300%"
```

## Integration with Dashboard

When using `--dashboard`, traffic patterns provide enhanced monitoring:

```bash
pipegen run --message-rate 100 --duration 10m \
  --traffic-pattern "2m-4m:400%,6m-8m:300%" \
  --dashboard
```

The dashboard shows:
- **Real-time rate changes** with visual indicators
- **Traffic pattern timeline** with current phase
- **Performance impact** during spikes
- **Resource utilization** trends
- **Error rates** correlation with load

## Use Cases

### 🧪 Load Testing
```bash
# Test system limits
pipegen run --message-rate 50 --duration 15m \
  --traffic-pattern "5m-10m:800%" \
  --dashboard
```

### 📊 Capacity Planning  
```bash
# Model expected growth
pipegen run --message-rate 100 --duration 20m \
  --traffic-pattern "5m-15m:200%" \
  --generate-report
```

### ⚡ Autoscaling Validation
```bash
# Verify scaling behavior
pipegen run --message-rate 25 --duration 12m \
  --traffic-pattern "2m-4m:600%,8m-10m:400%" \
  --dashboard
```

### 🔍 Failure Testing
```bash
# Stress test with extreme spikes
pipegen run --message-rate 30 --duration 8m \
  --traffic-pattern "2m-3m:1000%,5m-6m:800%" \
  --dashboard
```

## Troubleshooting

### Common Issues

**Pattern parsing errors:**
```bash
# ❌ Missing percentage sign
--traffic-pattern "30s-60s:300"

# ✅ Correct format  
--traffic-pattern "30s-60s:300%"
```

**Time format errors:**
```bash
# ❌ Invalid time format
--traffic-pattern "30sec-60sec:300%"

# ✅ Correct format
--traffic-pattern "30s-60s:300%"
```

### Performance Issues

If experiencing performance problems during high-rate spikes:

1. **Reduce spike magnitude**: Use lower multipliers
2. **Shorter spike duration**: Limit high-rate periods  
3. **Monitor resources**: Use `--dashboard` to track system impact
4. **Adjust batch size**: Modify producer settings

## Next Steps

- **[Dashboard](./dashboard)** - Monitor traffic patterns in real-time
- **[Examples](./examples)** - See traffic patterns in action
- **[Commands](./commands/run)** - Full `pipegen run` documentation  
- **[Performance Tuning](./advanced/performance)** - Optimize for high throughput
