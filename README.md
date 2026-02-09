# HyperFleet Credential Provider

> Multi-cloud Kubernetes authentication token provider for GKE, EKS, and AKS clusters

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

## Overview

HyperFleet Credential Provider is a standalone binary that generates short-lived Kubernetes authentication tokens for Google Kubernetes Engine (GKE), Amazon Elastic Kubernetes Service (EKS), and Azure Kubernetes Service (AKS) without requiring cloud CLI tools like `gcloud`, `aws`, or `az`.

It implements the [Kubernetes client-go credential plugin](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins) mechanism and is designed to work seamlessly with Prow CI workflows and other Kubernetes-native environments.

### Key Features

- **Multi-Cloud Support**: GCP (GKE), AWS (EKS), Azure (AKS)
- **No CLI Dependencies**: Pure Go implementation using cloud provider SDKs
- **Container-Ready**: Small container images (~40MB) with distroless base
- **Standard Interface**: Kubernetes exec plugin compatible
- **Prow CI Integration**: Purpose-built for CI/CD workflows

## Quick Start

### Installation

```bash
# Download binary (replace VERSION with latest release)
curl -L https://github.com/openshift-hyperfleet/hyperfleet-credential-provider/releases/download/VERSION/hyperfleet-credential-provider-linux-amd64 -o hyperfleet-credential-provider
chmod +x hyperfleet-credential-provider
sudo mv hyperfleet-credential-provider /usr/local/bin/

# Or build from source
git clone https://github.com/openshift-hyperfleet/hyperfleet-credential-provider.git
cd hyperfleet-credential-provider
make build
sudo cp bin/hyperfleet-credential-provider /usr/local/bin/
```

### Container Image

```bash
# Pull from registry
podman pull quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest

# Or build locally
make image
```

### Basic Usage

```bash
# Check version
hyperfleet-credential-provider version

# Generate GCP token
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
hyperfleet-credential-provider get-token \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --region=us-central1

# Generate kubeconfig for GKE
hyperfleet-credential-provider generate-kubeconfig \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --region=us-central1 \
  --credentials-file=/path/to/service-account.json \
  --output=kubeconfig.yaml
```

## Commands

### `get-token`

Generate a Kubernetes authentication token in ExecCredential format.

**Usage:**
```bash
hyperfleet-credential-provider get-token [flags]
```

**Flags:**
- `--provider` - Cloud provider (gcp, aws, azure) [required]
- `--cluster-name` - Cluster name [required]
- `--credentials-file` - Path to credentials file
- Provider-specific flags (see examples below)

**Examples:**

```bash
# GCP
hyperfleet-credential-provider get-token \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --region=us-central1

# AWS
hyperfleet-credential-provider get-token \
  --provider=aws \
  --cluster-name=my-cluster \
  --region=us-east-1

# Azure
hyperfleet-credential-provider get-token \
  --provider=azure \
  --cluster-name=my-cluster \
  --subscription-id=12345678-1234-1234-1234-123456789012 \
  --tenant-id=87654321-4321-4321-4321-210987654321
```

### `generate-kubeconfig`

Generate a complete kubeconfig file with exec plugin configuration.

**Usage:**
```bash
hyperfleet-credential-provider generate-kubeconfig [flags]
```

**Flags:**
- `--provider` - Cloud provider (gcp, aws, azure) [required]
- `--cluster-name` - Cluster name [required]
- `--output` - Output file path (default: stdout)
- `--credentials-file` - Path to credentials file
- Provider-specific flags

**Example:**
```bash
hyperfleet-credential-provider generate-kubeconfig \
  --provider=gcp \
  --cluster-name=hyperfleet-dev-prow \
  --project-id=my-project \
  --region=us-central1-a \
  --credentials-file=/vault/secrets/gcp-sa.json \
  --output=kubeconfig.yaml
```

### `get-cluster-info`

Get cluster information (endpoint, CA certificate).

**Usage:**
```bash
hyperfleet-credential-provider get-cluster-info [flags]
```

**Example:**
```bash
hyperfleet-credential-provider get-cluster-info \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --region=us-central1
```

## Environment Variables

All command-line flags can be set via environment variables using the prefix `HFCP_` followed by the flag name in uppercase with hyphens replaced by underscores.

**Priority order:** Command-line flags > Environment variables > Default values

### Supported Environment Variables

| Environment Variable | Flag Equivalent | Description |
|---------------------|----------------|-------------|
| `HFCP_LOG_LEVEL` | `--log-level` | Log level (debug, info, warn, error) |
| `HFCP_LOG_FORMAT` | `--log-format` | Log format (json, console) |
| `HFCP_CREDENTIALS_FILE` | `--credentials-file` | Path to credentials file |
| `HFCP_PROVIDER` | `--provider` | Cloud provider (gcp, aws, azure) |
| `HFCP_CLUSTER_NAME` | `--cluster-name` | Cluster name |
| `HFCP_REGION` | `--region` | Cloud region/location |
| `HFCP_PROJECT_ID` | `--project-id` | GCP project ID |
| `HFCP_ACCOUNT_ID` | `--account-id` | AWS account ID |
| `HFCP_SUBSCRIPTION_ID` | `--subscription-id` | Azure subscription ID |
| `HFCP_TENANT_ID` | `--tenant-id` | Azure tenant ID |
| `HFCP_RESOURCE_GROUP` | `--resource-group` | Azure resource group |
| `HFCP_TOKEN_DURATION` | `--token-duration` | Token duration (e.g., 1h, 30m) |

### Examples

**Using only environment variables:**

```bash
export HFCP_PROVIDER=gcp
export HFCP_CLUSTER_NAME=my-cluster
export HFCP_PROJECT_ID=my-project
export HFCP_REGION=us-central1
export HFCP_CREDENTIALS_FILE=/path/to/gcp-sa.json

# All flags are set via environment variables
hyperfleet-credential-provider generate-kubeconfig --output=kubeconfig.yaml
```

**Mixing environment variables and flags:**

```bash
# Set common values via environment variables
export HFCP_PROVIDER=gcp
export HFCP_CREDENTIALS_FILE=/path/to/gcp-sa.json

# Override or add specific values via flags
hyperfleet-credential-provider generate-kubeconfig \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --region=us-central1 \
  --output=kubeconfig.yaml
```

**Container environment (Prow CI):**

```yaml
containers:
- name: setup
  image: quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
  env:
  - name: HFCP_PROVIDER
    value: gcp
  - name: HFCP_PROJECT_ID
    value: my-project
  - name: HFCP_REGION
    value: us-central1
  - name: HFCP_CREDENTIALS_FILE
    value: /vault/secrets/gcp-sa.json
  command:
  - hyperfleet-credential-provider
  - generate-kubeconfig
  - --cluster-name=my-cluster
  - --output=/workspace/kubeconfig.yaml
```

## Configuration

### Google Cloud Platform (GKE)

**Prerequisites:**
- GCP service account with permissions:
  - `roles/container.clusterViewer`
  - `roles/iam.serviceAccountTokenCreator`
- Service account key file (JSON)

**Kubeconfig Example:**
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
      command: /usr/local/bin/hyperfleet-credential-provider
      args:
      - get-token
      - --provider=gcp
      - --cluster-name=my-cluster
      - --project-id=my-gcp-project
      - --region=us-central1
      env:
      - name: GOOGLE_APPLICATION_CREDENTIALS
        value: /vault/secrets/gcp-sa.json
      interactiveMode: Never
contexts:
- name: my-gke-context
  context:
    cluster: my-gke-cluster
    user: gke-user
current-context: my-gke-context
```

### Amazon Web Services (EKS)

**Prerequisites:**
- AWS IAM user or role with permissions:
  - `eks:DescribeCluster`
  - `sts:GetCallerIdentity`
- AWS credentials (access key or IAM role)

**Kubeconfig Example:**
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
      command: /usr/local/bin/hyperfleet-credential-provider
      args:
      - get-token
      - --provider=aws
      - --cluster-name=my-eks-cluster
      - --region=us-east-1
      env:
      - name: AWS_CREDENTIALS_FILE
        value: /vault/secrets/aws-credentials
      interactiveMode: Never
contexts:
- name: my-eks-context
  context:
    cluster: my-eks-cluster
    user: eks-user
current-context: my-eks-context
```

### Microsoft Azure (AKS)

**Prerequisites:**
- Azure service principal with permissions:
  - `Azure Kubernetes Service Cluster User Role`
- Service principal credentials

**Kubeconfig Example:**
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
      command: /usr/local/bin/hyperfleet-credential-provider
      args:
      - get-token
      - --provider=azure
      - --cluster-name=my-aks-cluster
      - --subscription-id=12345678-1234-1234-1234-123456789012
      - --tenant-id=87654321-4321-4321-4321-210987654321
      env:
      - name: AZURE_CREDENTIALS_FILE
        value: /vault/secrets/azure-credentials.json
      interactiveMode: Never
contexts:
- name: my-aks-context
  context:
    cluster: my-aks-cluster
    user: aks-user
current-context: my-aks-context
```

## Prow CI Integration

For Prow CI workflows, see the [Prow Integration Guide](docs/PROW_INTEGRATION_GUIDE.md).

**Quick Overview:**

1. **Deploy Pod**: Generate kubeconfig using `generate-kubeconfig` command
2. **Test Pod**: Use generated kubeconfig with kubectl
3. **Credentials**: Mount service account keys via Vault

**Example Deploy Pod:**
```bash
podman run --rm \
  -v /vault/secrets/gcp-sa.json:/vault/secrets/gcp-sa.json:ro \
  -v /workspace:/workspace \
  quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest \
  --credentials-file=/vault/secrets/gcp-sa.json \
  generate-kubeconfig \
  --provider=gcp \
  --cluster-name=prow-cluster \
  --project-id=my-project \
  --region=us-central1 \
  --output=/workspace/kubeconfig.yaml
```

## Development

### Prerequisites

- Go 1.24+
- Podman or Docker
- Make

### Building

```bash
# Build binary
make build

# Build container image
make image

# Run tests
make test

# Run linter
make lint
```

### Project Structure

```
hyperfleet-credential-provider/
├── cmd/provider/          # Main application entry point
│   ├── cluster/          # get-cluster-info command
│   ├── kubeconfig/       # generate-kubeconfig command
│   ├── token/            # get-token command
│   └── version/          # version command
├── internal/
│   ├── credentials/      # Credential loading
│   ├── execplugin/       # ExecCredential types
│   └── provider/         # Provider implementations
│       ├── gcp/         # GCP token generation
│       ├── aws/         # AWS token generation
│       └── azure/       # Azure token generation
├── pkg/
│   ├── logger/          # Structured logging
│   └── errors/          # Error types
├── examples/kubeconfig/  # Example kubeconfig files
├── test/integration/     # Integration tests
├── Dockerfile           # Multi-stage Docker build
└── Makefile            # Build automation
```

## Testing

```bash
# Run all unit tests
make test

# Run integration tests (requires cloud credentials)
make test-integration

# Generate coverage report
make coverage
```

## Troubleshooting

### Debug Mode

```bash
# Enable debug logging
hyperfleet-credential-provider get-token \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --log-level=debug

# Use console format for human-readable logs
hyperfleet-credential-provider get-token \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --log-level=debug \
  --log-format=console
```

### Common Issues

| Error | Cause | Solution |
|-------|-------|----------|
| `failed to load credentials` | Credential file not found | Verify file path and permissions |
| `failed to generate token` | Insufficient IAM permissions | Check cloud provider permissions |
| `failed to get cluster info` | Invalid cluster name/region | Verify cluster exists |
| `context deadline exceeded` | Network timeout | Check network connectivity |

## Security Model

1. **No Credential Persistence**: Credentials are read at runtime only
2. **Short-Lived Tokens**:
   - GCP: 1 hour
   - AWS: 15 minutes
   - Azure: 1 hour
3. **Least Privilege**: Minimal IAM permissions required
4. **Secure Container**: Distroless base image, non-root user

## Architecture

### Token Generation Flow

```
┌─────────┐
│ kubectl │
└────┬────┘
     │ 1. Reads kubeconfig
     │ 2. Detects exec plugin
     ▼
┌────────────────────────────────┐
│ hyperfleet-credential-provider │
│  ┌──────────────────────────┐  │
│  │ Load Credentials         │  │
│  └──────────┬───────────────┘  │
│  ┌──────────▼───────────────┐  │
│  │ Generate Cloud Token     │  │
│  │ (GCP/AWS/Azure SDK)      │  │
│  └──────────┬───────────────┘  │
│  ┌──────────▼───────────────┐  │
│  │ Return ExecCredential    │  │
│  └──────────────────────────┘  │
└────────────┬───────────────────┘
             │ 3. JSON output
     ┌───────▼────────┐
     │ kubectl        │
     │ Uses token for │
     │ API requests   │
     └────────────────┘
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `make test lint`
5. Submit a Pull Request

## License

Copyright 2026 Red Hat, Inc.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Links

- **GitHub**: https://github.com/openshift-hyperfleet/hyperfleet-credential-provider
- **Issues**: https://github.com/openshift-hyperfleet/hyperfleet-credential-provider/issues
- **Documentation**: See `docs/` directory
- **Examples**: See `examples/kubeconfig/` directory

---

**Part of the HyperFleet Project**
