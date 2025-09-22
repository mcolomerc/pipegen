# PipeGen

<p align="center">
  <a href="https://github.com/sponsors/mcolomerc" target="_blank">
    <img src="https://img.shields.io/badge/Sponsor-GitHub-blueviolet?logo=github" alt="Sponsor on GitHub"/>
  </a>
  <a href="https://buymeacoffee.com/mcolomerc" target="_blank">
    <img src="https://img.shields.io/badge/Buy%20Me%20a%20Coffee-mcolomerc-yellow?logo=buy-me-a-coffee" alt="Buy Me a Coffee"/>
  </a>
</p>

[PipeGen](https://mcolomerc.github.io/pipegen/) is a powerful CLI for creating and managing streaming data pipelines with Apache Kafka and FlinkSQL. It supports local development, AI-powered pipeline generation, and real-time monitoring. 

## Quick Start

```bash
# Install PipeGen
curl -sSL https://raw.githubusercontent.com/mcolomerc/pipegen/main/install.sh | bash

# Create an AI-generated fraud detection pipeline
pipegen init fraud-detection --describe "Monitor payment transactions, detect suspicious patterns using machine learning, and alert on potential fraud within 30 seconds"

# OR create from a CSV file with intelligent schema inference
pipegen init analytics-pipeline --input-csv ./data/user_events.csv

# Deploy local development stack
pipegen deploy

# Run with traffic spikes simulation
pipegen run --message-rate 100 --duration 10m --traffic-pattern "2m-4m:400%,6m-8m:300%" --dashboard
```

## Features ([see docs](https://mcolomerc.github.io/pipegen/features.html))

- 🚀 [Project scaffolding](https://mcolomerc.github.io/pipegen/features.html#project-scaffolding) - Generate complete project structure with SQL and AVRO schemas
- � **CSV Input Support** - Bootstrap pipelines from CSV files with intelligent schema inference and Docker integration
- �🐳 [Local development](https://mcolomerc.github.io/pipegen/features.html#local-development) - Docker Compose stack with Kafka, Flink, and Schema Registry
- 🤖 [AI-powered generation](https://mcolomerc.github.io/pipegen/ai-generation.html) - Describe your pipeline in natural language and let AI create optimized components
- 📊 [Smart producer](https://mcolomerc.github.io/pipegen/features.html#smart-producer) - Generate realistic test data matching any schema structure
- 👂 [Kafka consumer](https://mcolomerc.github.io/pipegen/features.html#kafka-consumer) - Validate pipeline output with built-in message validation
- ⚡ [FlinkSQL deployment](https://mcolomerc.github.io/pipegen/features.html#flinksql-deployment) - Deploy and manage FlinkSQL jobs locally or in the cloud
- 🏷️ [Dynamic resources](https://mcolomerc.github.io/pipegen/features.html#dynamic-resources) - Create unique topic names to avoid conflicts
- 🧹 [Auto cleanup](https://mcolomerc.github.io/pipegen/features.html#auto-cleanup) - Remove all created resources after testing
- ✅ [Validation](https://mcolomerc.github.io/pipegen/features.html#validation) - Validate project structure, SQL syntax, and AVRO schemas

---

For full documentation, advanced usage, and examples, visit: **[https://mcolomerc.github.io/pipegen/](https://mcolomerc.github.io/pipegen/)**

