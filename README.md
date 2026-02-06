# HyperFleet Cloud Provider

> Multi-cloud Kubernetes authentication token provider for GKE, EKS, and AKS clusters

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Test Status](https://img.shields.io/badge/tests-193%20passing-green.svg)]()
[![Coverage](https://img.shields.io/badge/coverage-90%25-green.svg)]()

## Overview

HyperFleet Cloud Provider is a standalone binary that generates short-lived Kubernetes authentication tokens for Google Kubernetes Engine (GKE), Amazon Elastic Kubernetes Service (EKS), and Azure Kubernetes Service (AKS) without requiring cloud CLI tools like `gcloud`, `aws`, or `az`.

It implements the [Kubernetes client-go credential plugin](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins) mechanism and is designed to work seamlessly with Prow CI workflows and other Kubernetes-native environments.

### Key Features

✅ **Multi-Cloud Support**
- Google Cloud Platform (GKE)
- Amazon Web Services (EKS)
- Microsoft Azure (AKS)

✅ **No CLI Dependencies**
- Pure Go implementation using cloud provider SDKs
- No need for `gcloud`, `aws`, or `az` CLI tools
- Smaller container images (~20-25MB) and faster execution

✅ **Production Ready**
- Comprehensive error handling and logging
- Health check endpoints for Kubernetes probes
- Prometheus metrics for monitoring
- OpenTelemetry distributed tracing
- Security-hardened Docker image (distroless base)

✅ **Easy Integration**
- Drop-in replacement for cloud CLI tools
- Standard Kubernetes exec plugin interface
- Works with any kubectl-compatible tool

✅ **Observability**
- 6 Prometheus metrics covering all operations
- OpenTelemetry distributed tracing
- Structured JSON logging
- Health and readiness probes

## Quick Start

### Installation

```bash
# Download binary (replace VERSION with latest release)
curl -L https://github.com/openshift-hyperfleet/hyperfleet-cloud-provider/releases/download/VERSION/hyperfleet-cloud-provider-linux-amd64 -o hyperfleet-cloud-provider
chmod +x hyperfleet-cloud-provider
sudo mv hyperfleet-cloud-provider /usr/local/bin/

# Or build from source
git clone https://github.com/openshift-hyperfleet/hyperfleet-cloud-provider.git
cd hyperfleet-cloud-provider
make build
sudo cp bin/hyperfleet-cloud-provider /usr/local/bin/
```

### Docker Image

```bash
# Pull from registry
docker pull ghcr.io/openshift-hyperfleet/hyperfleet-cloud-provider:latest

# Or build locally
make image
```

### Basic Usage

```bash
# Check version
hyperfleet-cloud-provider version

# Generate GCP token
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
hyperfleet-cloud-provider get-token \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project

# Generate AWS token
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
hyperfleet-cloud-provider get-token \
  --provider=aws \
  --cluster-name=my-cluster \
  --region=us-east-1

# Generate Azure token
export AZURE_CLIENT_ID=11111111-1111-1111-1111-111111111111
export AZURE_CLIENT_SECRET=your-client-secret
export AZURE_TENANT_ID=22222222-2222-2222-2222-222222222222
hyperfleet-cloud-provider get-token \
  --provider=azure \
  --cluster-name=my-cluster \
  --subscription-id=33333333-3333-3333-3333-333333333333
```

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
  - [Google Cloud Platform (GKE)](#google-cloud-platform-gke)
  - [Amazon Web Services (EKS)](#amazon-web-services-eks)
  - [Microsoft Azure (AKS)](#microsoft-azure-aks)
- [Kubernetes Integration](#kubernetes-integration)
- [Docker Deployment](#docker-deployment)
- [Health Endpoints](#health-endpoints)
- [Metrics & Observability](#metrics--observability)
- [Development](#development)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Architecture](#architecture)
- [Contributing](#contributing)

## Configuration

### Google Cloud Platform (GKE)

#### Prerequisites

1. GCP service account with permissions:
   - `roles/container.clusterViewer`
   - `roles/iam.serviceAccountTokenCreator`

2. Service account key file (JSON)

#### Kubeconfig Example

```yaml
apiVersion: v1
kind: Config
clusters:
- name: my-gke-cluster
  cluster:
    server: https://35.123.45.67
    certificate-authority-data: LS0tLS1CRUdJTi...
users:
- name: gke-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: hyperfleet-cloud-provider
      args:
      - get-token
      - --provider=gcp
      - --cluster-name=my-cluster
      - --project-id=my-gcp-project
      - --region=us-central1
      env:
      - name: GOOGLE_APPLICATION_CREDENTIALS
        value: /vault/secrets/gcp-sa.json
contexts:
- name: my-gke-context
  context:
    cluster: my-gke-cluster
    user: gke-user
current-context: my-gke-context
```

#### Environment Variables

```bash
# Required
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Optional
export LOG_LEVEL=info
export LOG_FORMAT=json
```

### Amazon Web Services (EKS)

#### Prerequisites

1. AWS IAM user or role with permissions:
   - `eks:DescribeCluster`
   - `sts:GetCallerIdentity`

2. AWS credentials (access key or IAM role)

#### Kubeconfig Example

```yaml
apiVersion: v1
kind: Config
clusters:
- name: my-eks-cluster
  cluster:
    server: https://ABC123.gr7.us-east-1.eks.amazonaws.com
    certificate-authority-data: LS0tLS1CRUdJTi...
users:
- name: eks-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: hyperfleet-cloud-provider
      args:
      - get-token
      - --provider=aws
      - --cluster-name=my-eks-cluster
      - --region=us-east-1
      env:
      - name: AWS_ACCESS_KEY_ID
        value: AKIAIOSFODNN7EXAMPLE
      - name: AWS_SECRET_ACCESS_KEY
        value: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
contexts:
- name: my-eks-context
  context:
    cluster: my-eks-cluster
    user: eks-user
current-context: my-eks-context
```

### Microsoft Azure (AKS)

#### Prerequisites

1. Azure service principal with permissions:
   - `Azure Kubernetes Service Cluster User Role`

2. Service principal credentials

#### Kubeconfig Example

```yaml
apiVersion: v1
kind: Config
clusters:
- name: my-aks-cluster
  cluster:
    server: https://my-aks-dns-12345678.hcp.eastus.azmk8s.io:443
    certificate-authority-data: LS0tLS1CRUdJTi...
users:
- name: aks-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: hyperfleet-cloud-provider
      args:
      - get-token
      - --provider=azure
      - --cluster-name=my-aks-cluster
      - --subscription-id=12345678-1234-1234-1234-123456789012
      - --tenant-id=87654321-4321-4321-4321-210987654321
      env:
      - name: AZURE_CLIENT_ID
        value: 11111111-1111-1111-1111-111111111111
      - name: AZURE_CLIENT_SECRET
        value: your-client-secret
contexts:
- name: my-aks-context
  context:
    cluster: my-aks-cluster
    user: aks-user
current-context: my-aks-context
```

## Kubernetes Integration

### Using with kubectl

Once your kubeconfig is configured with the exec plugin, kubectl commands work normally:

```bash
# Set kubeconfig
export KUBECONFIG=/path/to/kubeconfig.yaml

# Use kubectl
kubectl get nodes
kubectl get pods -A
kubectl apply -f deployment.yaml

# The provider is called automatically by kubectl
# Token is generated on-demand with each request
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hyperfleet-cloud-provider
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hyperfleet-cloud-provider
  template:
    metadata:
      labels:
        app: hyperfleet-cloud-provider
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
      containers:
      - name: provider
        image: ghcr.io/openshift-hyperfleet/hyperfleet-cloud-provider:latest
        ports:
        - name: health
          containerPort: 8080
        livenessProbe:
          httpGet:
            path: /healthz
            port: health
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /readyz
            port: health
          initialDelaySeconds: 5
          periodSeconds: 10
        resources:
          requests:
            cpu: 100m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 128Mi
```

## Docker Deployment

### Using Docker

```bash
# Run with GCP credentials
docker run --rm \
  -e GOOGLE_APPLICATION_CREDENTIALS=/vault/secrets/gcp-sa.json \
  -v /path/to/gcp-sa.json:/vault/secrets/gcp-sa.json:ro \
  ghcr.io/openshift-hyperfleet/hyperfleet-cloud-provider:latest \
  get-token \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project
```

### Using Docker Compose

```yaml
version: '3.8'

services:
  hyperfleet-cloud-provider:
    image: ghcr.io/openshift-hyperfleet/hyperfleet-cloud-provider:latest
    ports:
      - "8080:8080"
    environment:
      - LOG_LEVEL=info
      - TRACING_ENABLED=true
      - TRACING_ENDPOINT=jaeger:4317
    volumes:
      - ./credentials:/vault/secrets:ro

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "4317:4317"    # OTLP gRPC
```

## Health Endpoints

The provider exposes HTTP endpoints for Kubernetes liveness and readiness probes on port 8080.

### Available Endpoints

#### `GET /healthz` (Liveness Probe)

Indicates if the application is running.

**Response (200 OK):**
```json
{
  "status": "ok",
  "checks": {
    "server": "running"
  }
}
```

#### `GET /readyz` (Readiness Probe)

Indicates if the application is ready to serve traffic.

**Response (200 OK):**
```json
{
  "status": "ok",
  "checks": {
    "database": "ok",
    "api": "ok"
  }
}
```

#### `GET /metrics`

Prometheus metrics endpoint. See [Metrics & Observability](#metrics--observability).

#### `GET /`

Service information.

**Response:**
```json
{
  "service": "hyperfleet-cloud-provider",
  "status": "running",
  "uptime": "2h15m30s",
  "endpoints": ["/healthz", "/readyz", "/livez", "/metrics"]
}
```

## Metrics & Observability

### Prometheus Metrics

The provider exposes comprehensive Prometheus metrics on `/metrics`.

#### Available Metrics

1. **`hyperfleet_cloud_provider_token_requests_total`** (Counter)
   - Labels: `provider`, `status`
   - Total number of token generation requests

2. **`hyperfleet_cloud_provider_token_generation_duration_seconds`** (Histogram)
   - Labels: `provider`
   - Token generation duration in seconds

3. **`hyperfleet_cloud_provider_token_generation_errors_total`** (Counter)
   - Labels: `provider`, `error_type`
   - Total number of token generation errors

4. **`hyperfleet_cloud_provider_credential_validation_errors_total`** (Counter)
   - Labels: `provider`
   - Total number of credential validation errors

5. **`hyperfleet_cloud_provider_health_check_duration_seconds`** (Histogram)
   - Labels: `check_name`
   - Health check duration in seconds

6. **`hyperfleet_cloud_provider_health_check_errors_total`** (Counter)
   - Labels: `check_name`
   - Total number of health check errors

#### Example PromQL Queries

```promql
# Request rate by provider
sum(rate(hyperfleet_cloud_provider_token_requests_total[5m])) by (provider)

# Success rate percentage
(sum(rate(hyperfleet_cloud_provider_token_requests_total{status="success"}[5m])) / sum(rate(hyperfleet_cloud_provider_token_requests_total[5m]))) * 100

# P95 latency
histogram_quantile(0.95, rate(hyperfleet_cloud_provider_token_generation_duration_seconds_bucket[5m]))
```

### Kubernetes ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: hyperfleet-cloud-provider
spec:
  selector:
    matchLabels:
      app: hyperfleet-cloud-provider
  endpoints:
  - port: health
    path: /metrics
    interval: 10s
```

### OpenTelemetry Tracing

Distributed tracing with OpenTelemetry for debugging and performance analysis.

```bash
# Enable tracing
export TRACING_ENABLED=true
export TRACING_ENDPOINT=localhost:4317
export TRACING_SAMPLING_RATIO=0.1  # Sample 10% of traces
```

View traces in Jaeger UI at http://localhost:16686

## Development

### Prerequisites

- Go 1.24+
- Docker
- Make

### Building

```bash
# Build binary
make build

# Build for all platforms
make build-all

# Build Docker image
make image

# Run tests
make test

# Run linter
make lint
```

### Project Structure

```
hyperfleet-cloud-provider/
├── cmd/provider/          # Main application entry point
├── internal/
│   ├── config/           # Configuration loading
│   ├── provider/         # Provider implementations
│   │   ├── gcp/         # GCP token generation
│   │   ├── aws/         # AWS token generation
│   │   └── azure/       # Azure token generation
│   └── credentials/     # Credential loading
├── pkg/
│   ├── logger/          # Structured logging
│   ├── errors/          # Error types
│   ├── health/          # Health check server
│   ├── metrics/         # Prometheus metrics
│   └── tracing/         # OpenTelemetry tracing
├── Dockerfile           # Multi-stage Docker build
└── Makefile            # Build automation
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package tests
go test -v ./pkg/metrics/...
```

### Test Coverage

Current test coverage: **>90%** across all packages

- `cmd/provider`: ~85%
- `pkg/logger`: >90%
- `pkg/health`: >90%
- `pkg/metrics`: >95%
- `pkg/tracing`: >95%
- `internal/provider/gcp`: ~85%
- `internal/provider/aws`: ~90%
- `internal/provider/azure`: ~90%

**Total:** 193+ passing tests

## Troubleshooting

### Common Issues

#### Token Generation Fails

**Debug Steps:**
```bash
# Test token generation manually
hyperfleet-cloud-provider get-token \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --log-level=debug

# Validate credentials
hyperfleet-cloud-provider validate-credentials --provider=gcp
```

#### Health Check Failures

**Debug Steps:**
```bash
# Check health endpoint
kubectl exec -it <pod> -- wget -O- http://localhost:8080/readyz

# View logs
kubectl logs <pod>
```

#### High Latency

**Debug Steps:**
```bash
# Check metrics
curl http://localhost:8080/metrics | grep duration

# View traces in Jaeger UI
# Access at http://localhost:16686
```

### Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to load credentials` | Credential file not found or invalid | Verify file exists and permissions |
| `failed to generate token` | Insufficient permissions or API error | Check IAM permissions |
| `context deadline exceeded` | Network timeout or slow API | Check network connectivity |

### Logging

```bash
# Debug level (verbose)
export LOG_LEVEL=debug

# JSON format (for production)
export LOG_FORMAT=json

# Console format (for development)
export LOG_FORMAT=console
```

## Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│  ┌──────────────┐         ┌──────────────┐                 │
│  │   kubectl    │         │     Pod      │                 │
│  └──────┬───────┘         └──────┬───────┘                 │
│         │  Invokes               │  Invokes                 │
│         └────────────┬───────────┘                          │
└──────────────────────┼──────────────────────────────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │  hyperfleet-cloud-provider   │
        │  ┌────────────────────────┐  │
        │  │  Provider Router       │  │
        │  └───────┬────────────────┘  │
        │  ┌───────┴────────┐          │
        │  │ GCP │ AWS │ AZ │          │
        │  └───────┬────────┘          │
        │  ┌───────▼────────┐          │
        │  │ Credential     │          │
        │  │ Loader         │          │
        │  └────────────────┘          │
        └──────────────────────────────┘
                       │
                       ▼
        ┌──────────────────────────────┐
        │   Cloud Provider APIs        │
        └──────────────────────────────┘
```

### Token Generation Flow

1. kubectl reads kubeconfig
2. Detects exec plugin configuration
3. Invokes hyperfleet-cloud-provider
4. Provider loads credentials
5. Generates cloud-specific token
6. Returns ExecCredential JSON
7. kubectl uses token for API request

### Security Model

1. **No Credential Persistence** - Credentials read at runtime only
2. **Short-Lived Tokens** - GCP: 1h, AWS: 15m, Azure: 1h
3. **Least Privilege** - Minimal IAM permissions
4. **Secure Defaults** - Non-root user, distroless image

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run tests and linter (`make check`)
6. Open a Pull Request

## License

Copyright 2026 Red Hat, Inc.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Support

- **Issues:** https://github.com/openshift-hyperfleet/hyperfleet-cloud-provider/issues
- **Documentation:** See `docs/` directory
- **Examples:** See `configs/examples/` directory

---

**Made with ❤️ by the HyperFleet Team**
