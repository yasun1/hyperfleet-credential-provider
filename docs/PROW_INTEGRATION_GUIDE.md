# Prow CI Integration Guide

## Overview

This guide shows how to integrate `hyperfleet-credential-provider` into Prow CI workflows using a **two-stage approach**:

1. **Deploy Pod** - Setup phase (generates kubeconfig using single command)
2. **Test Pod** - Testing phase (uses kubeconfig with automatic token generation)

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    Prow CI Workflow                           │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  STAGE 1: Deploy Pod (Setup)                                │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Container: hyperfleet-credential-provider:latest (~25MB)    │ │
│  │                                                        │ │
│  │ 1. Read Vault-mounted credentials                     │ │
│  │    - GCP: /vault/secrets/gcp-sa.json                  │ │
│  │    - AWS: /vault/secrets/aws-credentials              │ │
│  │    - Azure: /vault/secrets/azure-credentials.json     │ │
│  │                                                        │ │
│  │ 2. Generate kubeconfig (single command!)              │ │
│  │    $ hyperfleet-credential-provider generate-kubeconfig \  │ │
│  │        --provider=$PROVIDER \                          │ │
│  │        --cluster-name=$CLUSTER \                       │ │
│  │        --output=/workspace/kubeconfig.yaml             │ │
│  │                                                        │ │
│  │    This command:                                      │ │
│  │    ✅ Fetches cluster info via cloud SDK              │ │
│  │    ✅ Generates kubeconfig with exec plugin           │ │
│  │    ✅ No additional scripts needed!                   │ │
│  │                                                        │ │
│  │ 3. Deploy HyperFleet components (optional)            │ │
│  │    $ kubectl apply -f manifests/                      │ │
│  └────────────────────────────────────────────────────────┘ │
│                          ↓                                   │
│              Share kubeconfig via /workspace                 │
│                          ↓                                   │
│  STAGE 2: Test Pod (Testing)                                │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Container: test-runner:latest (minimal image)          │ │
│  │                                                        │ │
│  │ 1. Use shared kubeconfig                              │ │
│  │    export KUBECONFIG=/workspace/kubeconfig.yaml       │ │
│  │                                                        │ │
│  │ 2. Run tests                                          │ │
│  │    $ kubectl get nodes                                 │ │
│  │    $ kubectl get pods -A                              │ │
│  │    $ make test-e2e                                    │ │
│  │                                                        │ │
│  │    Each kubectl call automatically triggers:          │ │
│  │    → hyperfleet-credential-provider get-token              │ │
│  │    → Fresh token generated (<300ms)                   │ │
│  │    → No cloud CLI tools needed!                       │ │
│  └────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────┘
```

## Quick Start Commands

### GCP/GKE
```bash
hyperfleet-credential-provider generate-kubeconfig \
  --provider=gcp \
  --cluster-name=my-cluster \
  --project-id=my-project \
  --region=us-central1-a \
  --output=/workspace/kubeconfig.yaml
```

### AWS/EKS
```bash
hyperfleet-credential-provider generate-kubeconfig \
  --provider=aws \
  --cluster-name=my-cluster \
  --region=us-east-1 \
  --output=/workspace/kubeconfig.yaml
```

### Azure/AKS
```bash
hyperfleet-credential-provider generate-kubeconfig \
  --provider=azure \
  --cluster-name=my-cluster \
  --subscription-id=<subscription-id> \
  --tenant-id=<tenant-id> \
  --resource-group=my-rg \
  --output=/workspace/kubeconfig.yaml
```

## Environment Variables

The `generate-kubeconfig` command automatically reads credentials from these environment variables:

### GCP
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to service account JSON file

### AWS
- `AWS_ACCESS_KEY_ID` - AWS access key ID
- `AWS_SECRET_ACCESS_KEY` - AWS secret access key
- `AWS_CREDENTIALS_FILE` - Path to credentials file (alternative)

### Azure
- `AZURE_CLIENT_ID` - Service principal client ID
- `AZURE_CLIENT_SECRET` - Service principal client secret
- `AZURE_TENANT_ID` - Azure tenant ID
- `AZURE_CREDENTIALS_FILE` - Path to credentials JSON file (alternative)

## ProwJob Configuration Examples

### Example 1: Simple GKE Test Job

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
        image: ghcr.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command:
        - hyperfleet-credential-provider
        - generate-kubeconfig
        - --provider=gcp
        - --cluster-name=hyperfleet-dev-prow
        - --project-id=hcm-hyperfleet
        - --region=us-central1-a
        - --output=/workspace/kubeconfig.yaml
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /vault/secrets/gcp-sa.json
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
        image: ghcr.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command:
        - hyperfleet-credential-provider
        - generate-kubeconfig
        - --provider=gcp
        - --cluster-name=hyperfleet-dev-prow
        - --project-id=hcm-hyperfleet
        - --region=us-central1-a
        - --output=/workspace/kubeconfig.yaml
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /vault/secrets/gcp-sa.json
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
        - {name: vault-secrets, mountPath: /vault/secrets}
      - name: test
        image: gcr.io/k8s-prow/test-runner:latest
        command: ["/test/run-e2e-tests.sh"]
        env:
        - name: KUBECONFIG
          value: /workspace/kubeconfig.yaml
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
        image: ghcr.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command:
        - hyperfleet-credential-provider
        - generate-kubeconfig
        - --provider=aws
        - --cluster-name=hyperfleet-dev-eks
        - --region=us-east-1
        - --output=/workspace/kubeconfig.yaml
        env:
        - name: AWS_CREDENTIALS_FILE
          value: /vault/secrets/aws-credentials
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
        - {name: vault-secrets, mountPath: /vault/secrets}
      - name: test
        image: gcr.io/k8s-prow/test-runner:latest
        command: ["/test/run-e2e-tests.sh"]
        env:
        - name: KUBECONFIG
          value: /workspace/kubeconfig.yaml
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
        image: ghcr.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command:
        - hyperfleet-credential-provider
        - generate-kubeconfig
        - --provider=azure
        - --cluster-name=hyperfleet-dev-aks
        - --subscription-id=$(AZURE_SUBSCRIPTION_ID)
        - --tenant-id=$(AZURE_TENANT_ID)
        - --resource-group=hyperfleet-rg
        - --output=/workspace/kubeconfig.yaml
        env:
        - name: AZURE_CREDENTIALS_FILE
          value: /vault/secrets/azure-credentials.json
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
        - {name: vault-secrets, mountPath: /vault/secrets}
      - name: test
        image: gcr.io/k8s-prow/test-runner:latest
        command: ["/test/run-e2e-tests.sh"]
        env:
        - name: KUBECONFIG
          value: /workspace/kubeconfig.yaml
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
        image: ghcr.io/openshift-hyperfleet/hyperfleet-credential-provider:latest
        command:
        - hyperfleet-credential-provider
        - generate-kubeconfig
        - --provider=gcp
        - --cluster-name=hyperfleet-dev-prow
        - --project-id=hcm-hyperfleet
        - --region=us-central1-a
        - --output=/workspace/kubeconfig.yaml
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /vault/secrets/gcp-sa.json
        volumeMounts:
        - {name: workspace, mountPath: /workspace}
        - {name: vault-secrets, mountPath: /vault/secrets}

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

          echo "✅ Deployment complete"
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
      - name: workspace
        emptyDir: {}
      - name: vault-secrets
        secret:
          secretName: hyperfleet-gcp-credentials
```

## Credentials Configuration

### Option 1: Kubernetes Secrets

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

### Option 2: Vault Integration (Recommended)

If using Vault for secret management, ensure the Deploy Pod has access to mount secrets at:

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

## Test Pod Examples

### Simple Test Script

```bash
#!/bin/bash
# Test Pod - Run E2E Tests

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
echo "✅ Cluster access verified"

# Run tests
echo ""
echo "Step 2: Running test suite..."
cd /workspace/src
go test -v ./test/e2e/... -timeout=30m

echo ""
echo "=========================================="
echo "✅ All tests passed!"
echo "=========================================="
```

## Summary

### Deploy Pod (Setup)

**Purpose:** Generate kubeconfig for cluster access
**Image:** `hyperfleet-credential-provider:latest` (~25MB)
**Runs:** Once per test workflow
**Outputs:** `kubeconfig.yaml` to shared volume

**Command:**
```bash
hyperfleet-credential-provider generate-kubeconfig \
  --provider=<gcp|aws|azure> \
  --cluster-name=<cluster> \
  --output=/workspace/kubeconfig.yaml \
  [provider-specific flags...]
```

### Test Pod (Testing)

**Purpose:** Run tests against cluster
**Image:** Your test image (minimal, no cloud CLIs needed)
**Runs:** After Deploy Pod completes
**Uses:** `kubeconfig.yaml` from shared volume

**Key Steps:**
1. Set `KUBECONFIG=/workspace/kubeconfig.yaml`
2. Run kubectl commands (auto-calls `get-token`)
3. Run test suite
4. Report results

### Benefits

✅ **No CLI tools in Test Pod** - Only `kubectl` needed
✅ **Single command setup** - Just `generate-kubeconfig`
✅ **Unified approach** - Same workflow for GCP, AWS, Azure
✅ **Small images** - Deploy: ~25MB, Test: depends on tests
✅ **Fast token generation** - <300ms per kubectl command
✅ **Secure** - Vault-mounted credentials, no hardcoding

## Troubleshooting

### Issue: "Failed to get cluster info"

**Solution:** Verify credentials are mounted correctly and have sufficient permissions.

```bash
# Check if credentials exist
ls -la /vault/secrets/

# Test credentials manually
hyperfleet-credential-provider validate-credentials --provider=gcp
```

### Issue: "kubectl: exec plugin failed"

**Solution:** Check that `hyperfleet-credential-provider` binary is in PATH in the Test Pod.

```bash
# Verify binary exists
which hyperfleet-credential-provider

# Check exec plugin logs (logs go to stderr)
kubectl get nodes 2>&1 | grep -i error
```

## Next Steps

1. Build and push the `hyperfleet-credential-provider` image
2. Create Kubernetes Secrets or configure Vault
3. Update your ProwJob configurations
4. Test with a simple job first
5. Roll out to all jobs

For detailed examples, see the [examples/kubeconfig/](../examples/kubeconfig/) directory.

For questions, see the main [README.md](../README.md) or open an issue.
