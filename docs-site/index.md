---
layout: home

hero:
  name: "PipeGen"
  text: "Streaming Data Pipeline Generator"
  tagline: "Create and manage Apache Kafka + FlinkSQL pipelines with AI-powered generation and real-time monitoring"
  image:
    src: /logo.png
    alt: PipeGen Logo
  actions:
    - theme: brand
      text: Get Started
      link: /getting-started
    - theme: alt
      text: View on GitHub
      link: https://github.com/mcolomerc/pipegen

features:
  - icon: 🚀
    title: Quick Project Scaffolding
    details: Generate complete pipeline projects with SQL statements, AVRO schemas, and Docker Compose setup in seconds.
    
  - icon: 🤖
    title: AI-Powered Generation
    details: Describe your pipeline in natural language and let AI create optimized FlinkSQL statements and schemas.
    
  - icon: 📊
    title: Dynamic Traffic Patterns
    details: Simulate realistic traffic spikes and load patterns for comprehensive testing and capacity planning.
    
  - icon: 🐳
    title: Local Development Stack
    details: Complete Docker-based development environment with Kafka, Flink, and Schema Registry.
    
  - icon: 📈
    title: Real-time Monitoring
    details: Live dashboard with WebSocket-based metrics, pipeline visualization, and performance analytics.
    
  - icon: 🏷️
    title: Dynamic Resource Management
    details: Automatic topic naming, schema registration, and cleanup to avoid conflicts in shared environments.
    
  - icon: ✅
    title: Comprehensive Validation
    details: Validate project structure, SQL syntax, AVRO schemas, and connectivity before deployment.
    
  - icon: 📄
    title: Execution Reports
    details: Generate detailed HTML reports with charts, metrics, and pipeline diagrams for analysis.
---

<div class="custom-container">

## Why PipeGen?

Building streaming data pipelines traditionally requires deep knowledge of Apache Kafka, FlinkSQL, AVRO schemas, and complex deployment configurations. **PipeGen eliminates this complexity** by providing:

- **🎯 Zero-config local development** - Complete stack with one command
- **🧠 AI-assisted pipeline creation** - Natural language to production-ready code
- **📊 Realistic testing capabilities** - Traffic pattern simulation for load testing
- **👀 Real-time visibility** - Live monitoring and comprehensive reporting
- **🔄 DevOps-ready workflows** - Automated deployment and cleanup

</div>

<div class="custom-container">

## Quick Example

```bash
# Install PipeGen
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash

# Create an AI-generated fraud detection pipeline
pipegen init fraud-detection --describe "Monitor payment transactions, detect suspicious patterns using machine learning, and alert on potential fraud within 30 seconds"

# Deploy local development stack
pipegen deploy

# Run with traffic spikes simulation
pipegen run --message-rate 100 --duration 10m --traffic-pattern "2m-4m:400%,6m-8m:300%" --dashboard
```

</div>

<div class="custom-container">

## Live Dashboard & Monitoring

<div style="text-align: center; margin: 2rem 0;">
  <img src="/screenshot.png" alt="PipeGen Dashboard" style="max-width: 100%; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.1);">
</div>

The integrated dashboard provides real-time visibility into your streaming pipeline with:

- **Live metrics updates** every second via WebSocket
- **Interactive pipeline flow** visualization  
- **Performance analytics** with latency percentiles and throughput trends
- **Error tracking** with resolution suggestions
- **Data quality monitoring** and schema validation
- **Exportable HTML reports** with comprehensive charts

</div>

<div class="custom-container">

## Perfect for Teams

<div class="grid-2">
  <div class="grid-item">
    <h3>🏢 Enterprises</h3>
    <ul>
      <li>Rapid prototyping of streaming solutions</li>
      <li>Load testing and capacity planning</li>
      <li>Training and onboarding new team members</li>
      <li>Standardized pipeline templates</li>
    </ul>
  </div>
  
  <div class="grid-item">
    <h3>👨‍💻 Developers</h3>
    <ul>
      <li>Learn Kafka and FlinkSQL hands-on</li>
      <li>Test streaming concepts locally</li>
      <li>Validate pipeline logic before production</li>
      <li>Generate boilerplate code quickly</li>
    </ul>
  </div>
</div>

</div>

<style>
.custom-container {
  max-width: 1152px;
  margin: 0 auto;
  padding: 2rem 1.5rem;
}

.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
  margin-top: 2rem;
}

.grid-item {
  padding: 1.5rem;
  border: 1px solid var(--vp-c-border);
  border-radius: 8px;
}

.grid-item h3 {
  margin-top: 0;
  color: var(--vp-c-brand-1);
}

@media (max-width: 768px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
}
</style>
