# Integration Tests

This directory contains integration tests that interact with real cloud providers (GCP, AWS, Azure).

## Prerequisites

Integration tests require:
1. Valid cloud credentials
2. Access to real Kubernetes clusters in each cloud
3. Appropriate IAM permissions

## Running Integration Tests

Integration tests are marked with the `integration` build tag and are skipped by default.

### Run All Integration Tests

```bash
go test -v -tags=integration ./test/integration/...
```

### Run Specific Provider Tests

```bash
# GCP only
go test -v -tags=integration ./test/integration/ -run TestGCP

# AWS only
go test -v -tags=integration ./test/integration/ -run TestAWS

# Azure only
go test -v -tags=integration ./test/integration/ -run TestAzure
```

### Using Makefile

```bash
make test-integration
```

## Environment Variables

### GCP Integration Tests

Required:
- `GOOGLE_APPLICATION_CREDENTIALS` - Path to GCP service account JSON file
- `GCP_TEST_PROJECT_ID` - GCP project ID
- `GCP_TEST_CLUSTER_NAME` - GKE cluster name
- `GCP_TEST_REGION` - GKE cluster location (region or zone)

Example:
```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json
export GCP_TEST_PROJECT_ID=my-gcp-project
export GCP_TEST_CLUSTER_NAME=my-gke-cluster
export GCP_TEST_REGION=us-central1-a

go test -v -tags=integration ./test/integration/ -run TestGCP
```

### AWS Integration Tests

Required:
- Credentials (choose one):
  - `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`
  - `AWS_CREDENTIALS_FILE` - Path to AWS credentials file
- `AWS_TEST_CLUSTER_NAME` - EKS cluster name
- `AWS_TEST_REGION` - AWS region

Optional:
- `AWS_SESSION_TOKEN` - For temporary credentials

Example:
```bash
export AWS_ACCESS_KEY_ID=AKIAxxxxxxxxxxxx
export AWS_SECRET_ACCESS_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export AWS_TEST_CLUSTER_NAME=my-eks-cluster
export AWS_TEST_REGION=us-east-1

go test -v -tags=integration ./test/integration/ -run TestAWS
```

### Azure Integration Tests

Required:
- Credentials (choose one):
  - `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, and `AZURE_TENANT_ID`
  - `AZURE_CREDENTIALS_FILE` - Path to Azure service principal JSON file
- `AZURE_TEST_CLUSTER_NAME` - AKS cluster name
- `AZURE_TEST_SUBSCRIPTION_ID` - Azure subscription ID
- `AZURE_TEST_RESOURCE_GROUP` - Resource group name

Example:
```bash
export AZURE_CLIENT_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
export AZURE_CLIENT_SECRET=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
export AZURE_TENANT_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
export AZURE_TEST_CLUSTER_NAME=my-aks-cluster
export AZURE_TEST_SUBSCRIPTION_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
export AZURE_TEST_RESOURCE_GROUP=my-resource-group

go test -v -tags=integration ./test/integration/ -run TestAzure
```

## What Tests Cover

Each provider's integration tests verify:

1. **Provider Creation**: Successfully create provider with credentials
2. **Token Generation**: Generate a valid authentication token
   - Token is not empty
   - Token has correct format (Bearer, k8s-aws-v1, etc.)
   - Token has valid expiration time
3. **Cluster Info Retrieval**: Get cluster details from the cloud API
   - Endpoint URL
   - CA certificate
   - Kubernetes version
   - Cloud-specific metadata (ARN, Resource ID, etc.)
4. **Credential Validation**: Validate that credentials work
5. **End-to-End Workflow**: Simulate complete generate-kubeconfig → kubectl flow
6. **Error Handling**: Test behavior with invalid credentials

## CI/CD Integration

In CI/CD pipelines:

```bash
# Set environment variables from secrets
export GOOGLE_APPLICATION_CREDENTIALS=${GCP_SA_KEY_FILE}
export GCP_TEST_PROJECT_ID=${GCP_PROJECT}
export GCP_TEST_CLUSTER_NAME=${GKE_CLUSTER}
export GCP_TEST_REGION=${GKE_REGION}

# Run integration tests
go test -v -tags=integration -timeout=10m ./test/integration/...
```

## Skipping Tests

Tests automatically skip if required environment variables are not set:

```
=== RUN   TestGCPIntegration
--- SKIP: TestGCPIntegration (0.00s)
    gcp_test.go:35: Skipping GCP integration test: missing required environment variables
```

## Troubleshooting

### Timeouts

Integration tests may take longer than the default 10-minute timeout:

```bash
go test -v -tags=integration -timeout=30m ./test/integration/...
```

### Rate Limiting

Cloud APIs may rate limit requests. Add delays between test runs if needed.

### Permissions

Ensure your service accounts/credentials have sufficient permissions:
- **GCP**: `container.clusters.get`, `iam.serviceAccounts.getAccessToken`
- **AWS**: `eks:DescribeCluster`, `sts:GetCallerIdentity`
- **Azure**: `Microsoft.ContainerService/managedClusters/read`, `Microsoft.ContainerService/managedClusters/listClusterUserCredential/action`

## Example Complete Test Run

```bash
# Set all environment variables
export GOOGLE_APPLICATION_CREDENTIALS=~/gcp-sa.json
export GCP_TEST_PROJECT_ID=my-project
export GCP_TEST_CLUSTER_NAME=test-cluster
export GCP_TEST_REGION=us-central1-a

# Run GCP integration tests
go test -v -tags=integration -run TestGCP ./test/integration/

# Expected output:
# === RUN   TestGCPIntegration
# === RUN   TestGCPIntegration/CreateProvider
# === RUN   TestGCPIntegration/GetToken
#     gcp_test.go:78: Token generated successfully, expires at: 2024-02-06T14:30:00Z
# === RUN   TestGCPIntegration/GetClusterInfo
#     gcp_test.go:102: Cluster info retrieved: endpoint=35.123.45.67, version=1.27.5-gke.200, location=us-central1-a
# === RUN   TestGCPIntegration/ValidateCredentials
# === RUN   TestGCPIntegration/EndToEnd
# --- PASS: TestGCPIntegration (5.23s)
```
