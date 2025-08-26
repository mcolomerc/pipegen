# pipegen deploy

The `deploy` command deploys your pipeline to a target environment using containerization.

## Usage

```bash
pipegen deploy [flags]
```

## Examples

```bash
# Deploy to local Docker
pipegen deploy --target local

# Deploy to Kubernetes
pipegen deploy --target k8s --namespace my-namespace

# Deploy with custom image tag
pipegen deploy --target k8s --image-tag v1.0.0

# Deploy with resource limits
pipegen deploy --cpu 2 --memory 4Gi
```

## Flags

- `--target` - Deployment target (local, k8s, docker-compose)
- `--namespace` - Kubernetes namespace (default: default)
- `--image-tag` - Docker image tag (default: latest)
- `--cpu` - CPU resource limit
- `--memory` - Memory resource limit
- `--replicas` - Number of replicas (default: 1)
- `--config` - Configuration file path
- `--dry-run` - Generate deployment files without deploying
- `--help` - Show help for deploy command

## Deployment Targets

### Local Docker
Deploys using Docker Compose for local development:

```bash
pipegen deploy --target local
```

Creates and runs:
- Kafka container
- Flink JobManager and TaskManager
- Your pipeline application

### Kubernetes
Deploys to a Kubernetes cluster:

```bash
pipegen deploy --target k8s --namespace production
```

Creates:
- Deployment manifests
- Service definitions
- ConfigMaps for configuration
- Persistent volumes if needed

### Docker Compose
Generates docker-compose.yml for custom deployments:

```bash
pipegen deploy --target docker-compose --dry-run
```

## Prerequisites

### For Local Deployment
- Docker installed and running
- Docker Compose available

### For Kubernetes Deployment
- kubectl configured and authenticated
- Sufficient cluster resources
- Required namespaces created

## Generated Resources

### Kubernetes Resources
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pipegen-pipeline
spec:
  replicas: 3
  selector:
    matchLabels:
      app: pipegen-pipeline
  template:
    metadata:
      labels:
        app: pipegen-pipeline
    spec:
      containers:
      - name: pipeline
        image: pipegen:latest
        resources:
          limits:
            cpu: 2
            memory: 4Gi
```

### Docker Compose
```yaml
version: '3.8'
services:
  pipeline:
    image: pipegen:latest
    environment:
      - KAFKA_BROKERS=kafka:9092
    depends_on:
      - kafka
      - flink-jobmanager
```

## Configuration

Deployment configuration can be customized in `deploy.yaml`:

```yaml
deployment:
  target: k8s
  namespace: production
  resources:
    cpu: "2"
    memory: "4Gi"
  replicas: 3
  
image:
  registry: myregistry.com
  repository: pipegen
  tag: v1.0.0
```

## Monitoring and Health Checks

Deployed pipelines include:

- **Health Checks**: HTTP endpoints for readiness/liveness
- **Metrics**: Prometheus metrics exposure
- **Logging**: Structured JSON logging
- **Tracing**: Distributed tracing support

## Scaling

### Manual Scaling
```bash
# Scale Kubernetes deployment
kubectl scale deployment pipegen-pipeline --replicas=5

# Scale Docker Compose
docker-compose up --scale pipeline=3
```

### Auto-scaling (Kubernetes)
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: pipegen-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: pipegen-pipeline
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## Troubleshooting

### Common Issues

**Insufficient Resources**
```
Error: pods "pipegen-pipeline-xxx" is forbidden: exceeded quota
Solution: Request more resources or reduce resource limits
```

**Image Pull Errors**
```
Error: ErrImagePull
Solution: Check image name, tag, and registry authentication
```

**Configuration Errors**
```
Error: configmap not found
Solution: Ensure configuration is properly deployed
```

## Rolling Updates

Update deployed pipelines without downtime:

```bash
# Update image
pipegen deploy --target k8s --image-tag v1.1.0

# Update configuration
kubectl apply -f updated-config.yaml
kubectl rollout restart deployment pipegen-pipeline
```

## See Also

- [Configuration](../configuration.md)
- [Dashboard](../dashboard.md)
- [pipegen check](./check.md)
