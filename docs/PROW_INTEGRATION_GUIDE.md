# Prow CI Integration Guide

## Overview

This guide demonstrates how to integrate `hyperfleet-credential-provider` into Prow CI workflows using a two-stage approach:

1. **Deploy Pod**: Generate kubeconfig using `generate-kubeconfig` command
2. **Test Pod**: Execute tests using the generated kubeconfig

## Architecture

### Prow CI Workflow

```
+----------------------------------------------------------------+
|                      Prow CI Workflow                          |
+----------------------------------------------------------------+
|                                                                |
|  STAGE 1: Deploy Pod (Setup)                                   |
|  +----------------------------------------------------------+  |
|  | Container: hyperfleet-credential-provider (~40MB)        |  |
|  |                                                          |  |
|  | 1. Read Vault-mounted credentials                        |  |
|  |    - GCP: /vault/secrets/gcp-sa.json                     |  |
|  |    - AWS: /vault/secrets/aws-credentials                 |  |
|  |    - Azure: /vault/secrets/azure-credentials.json        |  |
|  |                                                          |  |
|  | 2. Generate kubeconfig                                   |  |
|  |    $ hyperfleet-credential-provider generate-kubeconfig  |  |
|  |        --provider=$PROVIDER                              |  |
|  |        --cluster-name=$CLUSTER                           |  |
|  |        --credentials-file=/vault/secrets/...             |  |
|  |        --output=/workspace/kubeconfig.yaml               |  |
|  |                                                          |  |
|  |    Fetches cluster info via cloud SDK                    |  |
|  |    No additional scripts needed                          |  |
|  +----------------------------------------------------------+  |
|                            |                                   |
|                            v                                   |
|                Share kubeconfig via /workspace                 |
|                            |                                   |
|                            v                                   |
|  STAGE 2: Test Pod (Testing)                                   |
|  +----------------------------------------------------------+  |
|  | Container: test-runner                                   |  |
|  |                                                          |  |
|  | 1. Use shared kubeconfig                                 |  |
|  |    export KUBECONFIG=/workspace/kubeconfig.yaml          |  |
|  |                                                          |  |
|  | 2. Run tests                                             |  |
|  |    $ kubectl get nodes                                   |  |
|  |    $ kubectl get pods -A                                 |  |
|  |    $ make test-e2e                                       |  |
|  |                                                          |  |
|  |    Each kubectl call automatically triggers:             |  |
|  |    -> hyperfleet-credential-provider get-token           |  |
|  |    -> Token generation (<2s)                             |  |
|  |    -> No cloud CLI tools needed                          |  |
|  +----------------------------------------------------------+  |
+----------------------------------------------------------------+
```

## Quick Start

### GCP/GKE

```bash
# Deploy Pod: Generate kubeconfig
podman run --rm \
  -v /vault/secrets/gcp-sa.json:/vault/secrets/gcp-sa.json:ro \
  -v /workspace:/workspace \
  quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest \
  --credentials-file=/vault/secrets/gcp-sa.json \
  generate-kubeconfig \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --region=us-central1-a \
  --output=/workspace/kubeconfig.yaml

# Test Pod: Use kubeconfig
export KUBECONFIG=/workspace/kubeconfig.yaml
kubectl get nodes
kubectl get pods -A
```

### AWS/EKS

```bash
# Deploy Pod: Generate kubeconfig
podman run --rm \
  -v /vault/secrets/aws-credentials:/vault/secrets/aws-credentials:ro \
  -v /workspace:/workspace \
  quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest \
  --credentials-file=/vault/secrets/aws-credentials \
  generate-kubeconfig \
  --provider=aws \
  --cluster-name=my-cluster \
  --region=us-east-1 \
  --output=/workspace/kubeconfig.yaml

# Test Pod: Use kubeconfig
export KUBECONFIG=/workspace/kubeconfig.yaml
kubectl get nodes
kubectl get pods -A
```

### Azure/AKS

```bash
# Deploy Pod: Generate kubeconfig
podman run --rm \
  -v /vault/secrets/azure-credentials.json:/vault/secrets/azure-credentials.json:ro \
  -v /workspace:/workspace \
  quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest \
  --credentials-file=/vault/secrets/azure-credentials.json \
  generate-kubeconfig \
  --provider=azure \
  --cluster-name=my-cluster \
  --subscription-id=<subscription-id> \
  --tenant-id=<tenant-id> \
  --resource-group=my-rg \
  --output=/workspace/kubeconfig.yaml

# Test Pod: Use kubeconfig
export KUBECONFIG=/workspace/kubeconfig.yaml
kubectl get nodes
kubectl get pods -A
```

## Environment Variables in ProwJobs

All command-line flags can be set via environment variables using the `HFCP_` prefix.

> **Note**: For complete list of environment variables and usage details, see the [main README](../README.md#environment-variables).

### ProwJob YAML Example

In ProwJob configurations, you can use environment variables instead of flags for cleaner YAML:

```yaml
# Using environment variables (recommended for ProwJobs)
- name: setup
  image: quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
  env:
  - name: HFCP_PROVIDER
    value: gcp
  - name: HFCP_CLUSTER_NAME
    value: hyperfleet-dev-prow
  - name: HFCP_PROJECT_ID
    value: hcm-hyperfleet
  - name: HFCP_REGION
    value: us-central1-a
  - name: HFCP_CREDENTIALS_FILE
    value: /vault/secrets/gcp-sa.json
  command:
  - hyperfleet-credential-provider
  args:
  - generate-kubeconfig
  - --output=/workspace/kubeconfig.yaml
```

This is cleaner than passing all parameters as flags:

```yaml
# Using flags (verbose)
- name: setup
  image: quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
  command:
  - hyperfleet-credential-provider
  args:
  - generate-kubeconfig
  - --provider=gcp
  - --cluster-name=hyperfleet-dev-prow
  - --project-id=hcm-hyperfleet
  - --region=us-central1-a
  - --credentials-file=/vault/secrets/gcp-sa.json
  - --output=/workspace/kubeconfig.yaml
```

## ProwJob Configuration

### Example 1: Basic GKE Test Job

```yaml
# prow/jobs/hyperfleet-test-gke.yaml
presubmits:
  openshift-hyperfleet/hyperfleet:
  - name: pull-hyperfleet-e2e-gke
    always_run: true
    decorate: true
    labels:
      preset-gcp-credentials: "true"
    spec:
      containers:
      # Deploy Pod - Generate kubeconfig
      - name: setup
        image: quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command:
        - hyperfleet-credential-provider
        args:
        - --credentials-file=/vault/secrets/gcp-sa.json
        - generate-kubeconfig
        - --provider=gcp
        - --cluster-name=hyperfleet-dev-prow
        - --project-id=hcm-hyperfleet
        - --region=us-central1-a
        - --output=/workspace/kubeconfig.yaml
        volumeMounts:
        - name: workspace
          mountPath: /workspace
        - name: vault-secrets
          mountPath: /vault/secrets
          readOnly: true

      # Test Pod - Run tests
      - name: test
        image: gcr.io/k8s-prow/test-runner:latest
        command:
        - /bin/bash
        - -c
        - |
          export KUBECONFIG=/workspace/kubeconfig.yaml

          echo "Running E2E tests..."
          kubectl get nodes
          kubectl get pods -A

          # Run actual tests
          make test-e2e
        volumeMounts:
        - name: workspace
          mountPath: /workspace

      volumes:
      - name: workspace
        emptyDir: {}
      - name: vault-secrets
        secret:
          secretName: hyperfleet-gcp-credentials
```

### Example 2: Multi-Cloud Test Matrix

```yaml
# prow/jobs/hyperfleet-test-multicloud.yaml
presubmits:
  openshift-hyperfleet/hyperfleet:

  # GCP Test
  - name: pull-hyperfleet-e2e-gke
    always_run: true
    decorate: true
    labels:
      preset-gcp-credentials: "true"
    spec:
      containers:
      - name: setup
        image: quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command: [hyperfleet-credential-provider]
        args:
        - --credentials-file=/vault/secrets/gcp-sa.json
        - generate-kubeconfig
        - --provider=gcp
        - --cluster-name=hyperfleet-dev-prow
        - --project-id=hcm-hyperfleet
        - --region=us-central1-a
        - --output=/workspace/kubeconfig.yaml
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
        - {name: vault-secrets, mountPath: /vault/secrets, readOnly: true}
      - name: test
        image: gcr.io/k8s-prow/test-runner:latest
        command: ["/test/run-e2e-tests.sh"]
        env:
        - {name: KUBECONFIG, value: /workspace/kubeconfig.yaml}
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
      volumes:
      - {name: workspace, emptyDir: {}}
      - {name: vault-secrets, secret: {secretName: hyperfleet-gcp-credentials}}

  # AWS Test
  - name: pull-hyperfleet-e2e-eks
    always_run: true
    decorate: true
    labels:
      preset-aws-credentials: "true"
    spec:
      containers:
      - name: setup
        image: quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command: [hyperfleet-credential-provider]
        args:
        - --credentials-file=/vault/secrets/aws-credentials
        - generate-kubeconfig
        - --provider=aws
        - --cluster-name=hyperfleet-dev-eks
        - --region=us-east-1
        - --output=/workspace/kubeconfig.yaml
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
        - {name: vault-secrets, mountPath: /vault/secrets, readOnly: true}
      - name: test
        image: gcr.io/k8s-prow/test-runner:latest
        command: ["/test/run-e2e-tests.sh"]
        env:
        - {name: KUBECONFIG, value: /workspace/kubeconfig.yaml}
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
      volumes:
      - {name: workspace, emptyDir: {}}
      - {name: vault-secrets, secret: {secretName: hyperfleet-aws-credentials}}

  # Azure Test
  - name: pull-hyperfleet-e2e-aks
    always_run: true
    decorate: true
    labels:
      preset-azure-credentials: "true"
    spec:
      containers:
      - name: setup
        image: quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command: [hyperfleet-credential-provider]
        args:
        - --credentials-file=/vault/secrets/azure-credentials.json
        - generate-kubeconfig
        - --provider=azure
        - --cluster-name=hyperfleet-dev-aks
        - --subscription-id=$(AZURE_SUBSCRIPTION_ID)
        - --tenant-id=$(AZURE_TENANT_ID)
        - --resource-group=hyperfleet-rg
        - --output=/workspace/kubeconfig.yaml
        env:
        - name: AZURE_SUBSCRIPTION_ID
          valueFrom:
            secretKeyRef:
              name: azure-config
              key: subscription-id
        - name: AZURE_TENANT_ID
          valueFrom:
            secretKeyRef:
              name: azure-config
              key: tenant-id
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
        - {name: vault-secrets, mountPath: /vault/secrets, readOnly: true}
      - name: test
        image: gcr.io/k8s-prow/test-runner:latest
        command: ["/test/run-e2e-tests.sh"]
        env:
        - {name: KUBECONFIG, value: /workspace/kubeconfig.yaml}
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
      volumes:
      - {name: workspace, emptyDir: {}}
      - {name: vault-secrets, secret: {secretName: hyperfleet-azure-credentials}}
```

### Example 3: With Deployment Step

```yaml
# prow/jobs/hyperfleet-deploy-and-test.yaml
presubmits:
  openshift-hyperfleet/hyperfleet:
  - name: pull-hyperfleet-deploy-test
    always_run: true
    decorate: true
    labels:
      preset-gcp-credentials: "true"
    spec:
      containers:
      # Step 1: Generate kubeconfig
      - name: setup
        image: quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command: [hyperfleet-credential-provider]
        args:
        - --credentials-file=/vault/secrets/gcp-sa.json
        - generate-kubeconfig
        - --provider=gcp
        - --cluster-name=hyperfleet-dev-prow
        - --project-id=hcm-hyperfleet
        - --region=us-central1-a
        - --output=/workspace/kubeconfig.yaml
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
        - {name: vault-secrets, mountPath: /vault/secrets, readOnly: true}

      # Step 2: Deploy HyperFleet
      - name: deploy
        image: gcr.io/k8s-prow/kubectl:latest
        command:
        - /bin/bash
        - -c
        - |
          export KUBECONFIG=/workspace/kubeconfig.yaml

          echo "Deploying HyperFleet components..."
          kubectl create namespace hyperfleet-system --dry-run=client -o yaml | kubectl apply -f -
          kubectl apply -f /workspace/src/manifests/ -n hyperfleet-system

          echo "Waiting for deployments..."
          kubectl wait --for=condition=available --timeout=300s \
            deployment --all -n hyperfleet-system

          echo "Deployment complete"
          kubectl get pods -n hyperfleet-system
        volumeMounts:
        - {name: workspace, mountPath: /workspace}

      # Step 3: Run tests
      - name: test
        image: gcr.io/k8s-prow/test-runner:latest
        command:
        - /bin/bash
        - -c
        - |
          export KUBECONFIG=/workspace/kubeconfig.yaml

          echo "Running E2E tests..."
          make test-e2e
        volumeMounts:
        - {name: workspace, mountPath: /workspace}

      volumes:
      - {name: workspace, emptyDir: {}}
      - {name: vault-secrets, secret: {secretName: hyperfleet-gcp-credentials}}
```

## Credentials Configuration

### Kubernetes Secrets

```yaml
# GCP Credentials
apiVersion: v1
kind: Secret
metadata:
  name: hyperfleet-gcp-credentials
  namespace: default
type: Opaque
data:
  gcp-sa.json: <base64-encoded-service-account-key>

---
# AWS Credentials
apiVersion: v1
kind: Secret
metadata:
  name: hyperfleet-aws-credentials
  namespace: default
type: Opaque
data:
  aws-credentials: <base64-encoded-aws-credentials-file>

---
# Azure Credentials
apiVersion: v1
kind: Secret
metadata:
  name: hyperfleet-azure-credentials
  namespace: default
type: Opaque
data:
  azure-credentials.json: <base64-encoded-azure-credentials-file>
```

### Vault Integration (Recommended)

Mount secrets at:
- GCP: `/vault/secrets/gcp-sa.json`
- AWS: `/vault/secrets/aws-credentials`
- Azure: `/vault/secrets/azure-credentials.json`

Example Vault annotations:

```yaml
annotations:
  vault.hashicorp.com/agent-inject: "true"
  vault.hashicorp.com/agent-inject-secret-gcp-sa.json: "secret/data/hyperfleet/gcp"
  vault.hashicorp.com/agent-inject-template-gcp-sa.json: |
    {{- with secret "secret/data/hyperfleet/gcp" -}}
    {{ .Data.data.credentials }}
    {{- end }}
  vault.hashicorp.com/role: "hyperfleet-prow"
```

## Test Script Example

```bash
#!/bin/bash
# Test Pod - E2E Tests

set -e

KUBECONFIG="${KUBECONFIG:-/workspace/kubeconfig.yaml}"

echo "=========================================="
echo "Running E2E Tests"
echo "=========================================="
echo "Kubeconfig: $KUBECONFIG"
echo "=========================================="

# Verify cluster access
echo ""
echo "Step 1: Verifying cluster access..."
kubectl cluster-info
kubectl get nodes -o wide
echo "Cluster access verified"

# Run tests
echo ""
echo "Step 2: Running test suite..."
cd /workspace/src
go test -v ./test/e2e/... -timeout=30m

echo ""
echo "=========================================="
echo "All tests passed"
echo "=========================================="
```

## Summary

### Deploy Pod (Setup)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Generate kubeconfig for cluster access |
| **Image** | `quay.io/openshift-hyperfleet/hyperfleet-credential-provider:latest` (~40MB) |
| **Frequency** | Once per test workflow |
| **Output** | `kubeconfig.yaml` to shared volume |

**Command:**
```bash
hyperfleet-credential-provider \
  --credentials-file=/vault/secrets/... \
  generate-kubeconfig \
  --provider=<gcp|aws|azure> \
  --cluster-name=<cluster> \
  --output=/workspace/kubeconfig.yaml \
  [provider-specific flags...]
```

### Test Pod (Testing)

| Attribute | Value |
|-----------|-------|
| **Purpose** | Run tests against cluster |
| **Image** | Your test image (no cloud CLIs needed) |
| **Frequency** | After Deploy Pod completes |
| **Input** | `kubeconfig.yaml` from shared volume |

**Steps:**
1. Set `KUBECONFIG=/workspace/kubeconfig.yaml`
2. Run kubectl commands (auto-calls `get-token`)
3. Execute test suite
4. Report results

### Benefits

- **No CLI Tools in Test Pod**: Only `kubectl` required
- **Single Command Setup**: Just `generate-kubeconfig`
- **Unified Approach**: Same workflow for GCP, AWS, Azure
- **Small Images**: Deploy ~40MB, Test depends on your tests
- **Fast Token Generation**: <2 seconds per kubectl command
- **Secure**: Vault-mounted credentials, no hardcoding

## Troubleshooting

### Issue: "Failed to get cluster info"

**Solution**: Verify credentials are mounted correctly and have sufficient permissions.

```bash
# Check if credentials exist
ls -la /vault/secrets/

# Test credential loading
hyperfleet-credential-provider get-cluster-info \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --region=us-central1 \
  --credentials-file=/vault/secrets/gcp-sa.json \
  --log-level=debug
```

### Issue: "kubectl: exec plugin failed"

**Solution**: Ensure `hyperfleet-credential-provider` binary is in PATH.

```bash
# Verify binary exists in Test Pod
which hyperfleet-credential-provider

# Check exec plugin logs (stderr)
kubectl get nodes 2>&1 | grep -i error
```

### Issue: "context deadline exceeded"

**Solution**: Check network connectivity and timeout settings.

```bash
# Increase timeout
kubectl get nodes --request-timeout=30s

# Check logs with debug level
hyperfleet-credential-provider get-token \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --log-level=debug
```

## Next Steps

1. Build and push `hyperfleet-credential-provider` image
2. Create Kubernetes Secrets or configure Vault
3. Update ProwJob configurations
4. Test with a simple job first
5. Roll out to all jobs

## Additional Resources

- [Main README](../README.md)
- [Example Kubeconfig Files](../examples/kubeconfig/)
- [GitHub Issues](https://github.com/openshift-hyperfleet/hyperfleet-credential-provider/issues)
