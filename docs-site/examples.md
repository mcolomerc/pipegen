# Examples

Explore real-world use cases and see PipeGen in action across different industries and scenarios.

## Quick Examples

### Basic Pipeline
```bash
# Create and run a basic analytics pipeline
pipegen init basic-analytics
cd basic-analytics
pipegen deploy
pipegen run --dashboard
```

### Load Testing
```bash
# Simulate Black Friday traffic
pipegen run --message-rate 100 --duration 10m \
  --traffic-pattern "2m-3m:500%,5m-7m:400%,8m-9m:600%" \
  --dashboard --generate-report
```

### AI-Generated Pipeline
```bash
# Let AI create a fraud detection system
pipegen init fraud-system --describe \
  "Monitor credit card transactions, detect suspicious patterns using statistical analysis, and alert on potential fraud within 30 seconds" \
  --domain "fintech"
```

## Industry Use Cases

### 🛍️ E-commerce & Retail

**Real-Time Product Recommendations**
```bash
pipegen init product-recommendations --describe \
  "Track user interactions with products, calculate item similarity scores, maintain user preference profiles, and generate personalized recommendations in real-time" \
  --domain "ecommerce"
```

**Inventory Management**
```bash
pipegen init inventory-alerts --describe \
  "Monitor product inventory levels across warehouses, track sales velocity, predict stockouts, and trigger reorder alerts when inventory falls below safety thresholds" \
  --domain "ecommerce"
```

**Customer Journey Analytics**
```bash
pipegen init customer-journey --describe \
  "Analyze customer touchpoints across web and mobile, track conversion funnels from awareness to purchase, identify drop-off points, and calculate attribution models" \
  --domain "ecommerce"
```

### 💰 Financial Services

**Fraud Detection System**
```bash
pipegen init fraud-detection --describe \
  "Analyze payment transactions in real-time, compare against customer spending patterns, detect anomalies using machine learning models, and flag suspicious transactions scoring above 0.85 confidence" \
  --domain "fintech"
```

**High-Frequency Trading**
```bash
pipegen init hft-signals --describe \
  "Process market data feeds from multiple exchanges, calculate technical indicators like MACD and RSI, identify arbitrage opportunities, and generate trading signals with microsecond latency" \
  --domain "fintech"
```

**Risk Management**
```bash
pipegen init risk-monitoring --describe \
  "Monitor trading positions in real-time, calculate VaR and portfolio exposure, track market volatility indicators, and trigger risk alerts when limits are exceeded" \
  --domain "fintech"
```

### 🏭 Manufacturing & IoT

**Predictive Maintenance**
```bash
pipegen init predictive-maintenance --describe \
  "Monitor industrial equipment sensors for temperature, vibration, and pressure, detect anomaly patterns, predict equipment failures using historical data, and schedule maintenance before breakdowns occur" \
  --domain "manufacturing"
```

**Quality Control**
```bash
pipegen init quality-monitoring --describe \
  "Process real-time production line data, monitor product quality metrics, detect defects using statistical process control, and trigger quality alerts when tolerances are exceeded" \
  --domain "manufacturing"
```

**Supply Chain Optimization**
```bash
pipegen init supply-chain --describe \
  "Track shipments across global supply chain, monitor delivery performance, predict delays based on weather and traffic data, and optimize routing for cost and time efficiency" \
  --domain "logistics"
```

### 🎮 Gaming & Entertainment

**Player Behavior Analytics**
```bash
pipegen init player-analytics --describe \
  "Track player actions and progression in real-time, identify engagement patterns, predict player churn, and generate personalized offers to improve retention" \
  --domain "gaming"
```

**Dynamic Game Balancing**
```bash
pipegen init game-balance --describe \
  "Monitor gameplay metrics like win rates and match duration, analyze player skill distribution, detect balance issues, and recommend game parameter adjustments" \
  --domain "gaming"
```

**Live Tournament Analytics**
```bash
pipegen init tournament-stats --describe \
  "Process live tournament data including player performance, match statistics, audience engagement, and generate real-time leaderboards and analytics dashboards" \
  --domain "gaming"
```

### 🏥 Healthcare & Life Sciences

**Patient Monitoring**
```bash
pipegen init patient-monitoring --describe \
  "Process vital signs from patient monitors including heart rate, blood pressure, and oxygen levels, detect critical value changes, and trigger immediate alerts for medical staff" \
  --domain "healthcare"
```

**Drug Discovery Analytics**
```bash
pipegen init drug-discovery --describe \
  "Analyze molecular compound data from research experiments, identify promising candidates based on efficacy and safety profiles, and track clinical trial progress" \
  --domain "healthcare"
```

### 📱 Social Media & Content

**Content Moderation**
```bash
pipegen init content-moderation --describe \
  "Analyze user-generated content in real-time, detect harmful or inappropriate material using NLP models, calculate toxicity scores, and flag content for human review" \
  --domain "social"
```

**Trend Detection**
```bash
pipegen init trend-detection --describe \
  "Monitor social media posts and engagement metrics, identify viral content and emerging trends, track hashtag popularity, and generate real-time trend reports" \
  --domain "social"
```

## Traffic Pattern Examples

### Black Friday Simulation
```bash
# E-commerce traffic during major sales event
pipegen run --message-rate 200 --duration 30m \
  --traffic-pattern "5m-10m:400%,15m-20m:600%,25m-28m:500%" \
  --dashboard
```

### Breaking News Event
```bash
# News website traffic spike
pipegen run --message-rate 100 --duration 15m \
  --traffic-pattern "2m-8m:800%,10m-12m:300%" \
  --dashboard --generate-report
```

### Business Hours Pattern
```bash
# Gradual increase during business hours
pipegen run --message-rate 50 --duration 20m \
  --traffic-pattern "2m-4m:150%,6m-8m:200%,10m-12m:300%,14m-16m:250%,18m-20m:150%" \
  --dashboard
```

### Gaming Event Launch
```bash
# Massive spike when new game features launch
pipegen run --message-rate 75 --duration 12m \
  --traffic-pattern "3m-8m:700%,10m-12m:400%" \
  --dashboard
```

## Testing Scenarios

### Performance Benchmarking
```bash
# Establish baseline performance
pipegen run --message-rate 100 --duration 10m --generate-report

# Test with 2x load
pipegen run --message-rate 200 --duration 10m --generate-report

# Test with extreme spikes
pipegen run --message-rate 100 --duration 10m \
  --traffic-pattern "2m-4m:1000%" --dashboard
```

### Failure Testing
```bash
# Test recovery from high load
pipegen run --message-rate 50 --duration 8m \
  --traffic-pattern "2m-3m:2000%,5m-6m:1500%" \
  --dashboard

# Sustained high load
pipegen run --message-rate 100 --duration 10m \
  --traffic-pattern "2m-8m:500%" \
  --dashboard --generate-report
```

### Capacity Planning
```bash
# Model expected growth over time
pipegen run --message-rate 100 --duration 20m \
  --traffic-pattern "5m-15m:200%" \
  --dashboard --generate-report

# Test autoscaling response
pipegen run --message-rate 50 --duration 15m \
  --traffic-pattern "3m-5m:400%,8m-10m:600%,12m-14m:300%" \
  --dashboard
```

## Integration Examples

### CI/CD Pipeline Testing
```bash
#!/bin/bash
# ci-test.sh - Automated pipeline testing

set -e

echo "🚀 Starting pipeline validation..."

# Create test pipeline
pipegen init ci-test-$(date +%s) --describe "Simple analytics for CI testing"

cd ci-test-*

# Validate project
pipegen validate

# Deploy local stack
pipegen deploy --startup-timeout 2m

# Run with traffic patterns and generate report
pipegen run --message-rate 100 --duration 3m \
  --traffic-pattern "30s-90s:300%" \
  --generate-report --reports-dir ./ci-reports

# Check for errors in report
if grep -q "ERROR" ci-reports/*.html; then
  echo "❌ Pipeline test failed - errors detected"
  exit 1
fi

echo "✅ Pipeline test passed"
```

### Monitoring Integration
```bash
# Export metrics to external monitoring
pipegen run --dashboard --message-rate 100 --duration 10m &
PIPELINE_PID=$!

# Collect metrics every 30 seconds
while kill -0 $PIPELINE_PID 2>/dev/null; do
  curl -s http://localhost:3000/api/metrics | \
    jq '.producer.rate' | \
    curl -X POST https://monitoring.company.com/metrics \
    -H "Content-Type: application/json" -d @-
  sleep 30
done
```

### Multi-Environment Testing
```bash
# Test across different configurations
environments=("local" "staging" "production")

for env in "${environments[@]}"; do
  echo "🌍 Testing in $env environment"
  
  pipegen run --config ".pipegen-$env.yaml" \
    --message-rate 100 --duration 5m \
    --traffic-pattern "1m-2m:200%,3m-4m:300%" \
    --generate-report --reports-dir "./reports-$env"
done
```

## Advanced Use Cases

### Multi-Pipeline Orchestration
```bash
# Run multiple pipelines simultaneously
pipegen run --dashboard-port 3001 --project-dir ./pipeline-1 &
pipegen run --dashboard-port 3002 --project-dir ./pipeline-2 &
pipegen run --dashboard-port 3003 --project-dir ./pipeline-3 &

# Aggregate monitoring
pipegen dashboard --port 3000 --aggregate-ports 3001,3002,3003
```

### Custom Schema Testing
```bash
# Test with real production schemas
pipegen init schema-validation \
  --input-schema ./production-events.avsc \
  --describe "Validate production event processing"

pipegen run --message-rate 1000 --duration 10m \
  --dashboard --generate-report
```

### Performance Profiling
```bash
# Profile different message rates
rates=(50 100 200 500 1000)

for rate in "${rates[@]}"; do
  echo "📊 Testing at $rate msg/sec"
  
  pipegen run --message-rate $rate --duration 2m \
    --generate-report --reports-dir "./profile-reports" \
    --traffic-pattern "30s-90s:200%"
done
```

## Sample Outputs

### Dashboard Metrics
When running with `--dashboard`, you'll see real-time metrics like:

- **Producer Rate**: 847 msg/sec (↑ in traffic spike)
- **Consumer Rate**: 845 msg/sec (lag: 2 messages)
- **End-to-End Latency**: P95 = 12ms, P99 = 28ms
- **Error Rate**: 0.02% (2 errors in last minute)
- **System Health**: ✅ All components healthy

### Report Generation
Generated HTML reports include:

- **Executive Summary**: Pipeline completed successfully, processed 50,000 messages
- **Performance Charts**: Throughput trends, latency percentiles, error rates
- **Traffic Analysis**: Pattern compliance, peak performance metrics
- **Resource Utilization**: CPU, memory, network usage during execution
- **Recommendations**: Optimal configurations, scaling suggestions

## Common Patterns

### Development Workflow
```bash
# 1. Create and validate
pipegen init my-pipeline --describe "..."
pipegen validate --check-connectivity

# 2. Quick test
pipegen run --message-rate 10 --duration 1m

# 3. Load test
pipegen run --message-rate 100 --duration 5m \
  --traffic-pattern "1m-2m:300%" --dashboard

# 4. Generate documentation
pipegen run --generate-report --duration 10m
```

### Production Validation
```bash
# 1. Baseline test
pipegen run --message-rate 100 --duration 10m --generate-report

# 2. Stress test with spikes  
pipegen run --message-rate 100 --duration 15m \
  --traffic-pattern "3m-5m:500%,8m-10m:700%,12m-14m:400%" \
  --dashboard --generate-report

# 3. Sustained load test
pipegen run --message-rate 200 --duration 30m --generate-report
```

## Next Steps

- **[Getting Started](./getting-started)** - Create your first pipeline
- **[AI Generation](./ai-generation)** - Use AI to generate pipelines
- **[Traffic Patterns](./traffic-patterns)** - Master load testing techniques
- **[Dashboard](./dashboard)** - Explore monitoring capabilities
- **[Commands](./commands)** - Learn all available commands
