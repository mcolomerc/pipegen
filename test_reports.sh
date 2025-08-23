#!/bin/bash

# Simple test to demonstrate report generation
cd /workspaces/pipegen

echo "🚀 Testing pipegen with report generation..."
echo "Creating test reports directory..."

mkdir -p /tmp/test-pipeline/reports

echo "Running pipegen with dry-run and report generation..."
./pipegen run \
  --project-dir /tmp/test-pipeline \
  --message-rate 500 \
  --duration 30s \
  --dry-run \
  --generate-report \
  --reports-dir /tmp/test-pipeline/reports

echo "📊 Checking generated reports..."
ls -la /tmp/test-pipeline/reports/

echo "🎯 Report generation complete!"
