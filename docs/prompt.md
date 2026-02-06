You are an AI system architect and senior Go engineer tasked with designing and implementing a multi-cloud Kubernetes credential provider runtime for Hyperfleet E2E CI workflows.

----------------------------------------
BACKGROUND
----------------------------------------

We run E2E tests using Prow CI.

Workflow:

1. Deploy Step
   - Uses a heavy container image with cloud CLIs
   - Deploys Hyperfleet components to Kubernetes cluster
   - Generates kubeconfig used by later CI steps

2. Test Step
   - Uses a lightweight container image
   - Contains only kubectl + test binaries
   - Must securely access Kubernetes cluster
   - Cannot depend on cloud CLIs

Credentials:

- Cloud service account credentials are stored in Vault
- Vault secrets are mounted directly into CI Pods
- Each CI step runs in a separate Pod
- Target clusters are long-lived clusters

----------------------------------------
SECURITY CONSTRAINTS
----------------------------------------

1. Deploy Step MUST NOT generate or share kubeconfig files containing long-lived authentication tokens.

2. Deploy Step may generate kubeconfig containing:
   - Cluster endpoint
   - Certificate authority data
   - Exec plugin configuration pointing to provider runtime

3. Test Step must obtain authentication tokens dynamically at runtime via the provider runtime.

4. Cloud credentials stored in Vault are considered long-lived and must never be copied into images or persisted into shared kubeconfig tokens.

----------------------------------------
PRIMARY OBJECTIVE
----------------------------------------

Design and implement a standalone binary:

    hyperfleet-credential-provider

This binary must:

1. Generate short-lived Kubernetes access tokens at runtime
2. Support multiple cloud providers:
   - GKE
   - EKS
   - AKS
3. Be compatible with Kubernetes exec authentication plugin
4. Work without cloud CLIs
5. Read credentials from Vault-mounted files
6. Be reusable across Deploy and Test steps
7. Follow least privilege and secure token generation practices

----------------------------------------
FUNCTIONAL REQUIREMENTS
----------------------------------------

### 1. Exec Plugin Compatibility

The provider must output credentials using:

client.authentication.k8s.io/v1 ExecCredential

Including:
- status.token
- expirationTimestamp

### 2. Provider Selection

Provider must be selectable via:
- kubeconfig exec arguments
- environment variables
- configuration file

### 3. Credential Sources

Support:

- GCP Service Account JSON
- AWS Access Key + Secret
- Azure Service Principal credentials

Credentials will be supplied via Vault-mounted files or environment variables.

### 4. Token Generation

Implement SDK-based token generation without cloud CLIs.

GKE:
- Generate IAM token compatible with GKE authentication

EKS:
- Generate STS presigned token compatible with aws-iam-authenticator

AKS:
- Generate Azure AD token compatible with AKS authentication

### 5. Runtime Behavior

Provider must:

- Generate tokens only when invoked by kubectl exec plugin
- Never store tokens on disk
- Validate token expiration
- Fail safely on credential errors

----------------------------------------
ARCHITECTURE REQUIREMENTS
----------------------------------------

Design must include:

- Pluggable provider interface
- Separation of responsibilities:
    - Credential loading
    - Cloud-specific token generation
    - Exec plugin output formatting
- Extensible provider registry

----------------------------------------
GO IMPLEMENTATION REQUIREMENTS
----------------------------------------

Claude must:

1. Design package structure
2. Provide production-ready Go code
3. Include context-based API design
4. Include robust error handling
5. Include unit testing scaffolding
6. Follow Kubernetes and Go best practices

----------------------------------------
EXPECTED PACKAGE STRUCTURE
----------------------------------------

Suggested layout:

- cmd/
- pkg/provider/
- pkg/provider/gcp/
- pkg/provider/aws/
- pkg/provider/azure/
- pkg/credentials/
- pkg/execplugin/
- pkg/config/

----------------------------------------
CI INTEGRATION REQUIREMENTS
----------------------------------------

Provider must support:

- Running inside Prow Pods
- Reading Vault-mounted credentials
- Running in minimal container images
- Being automatically invoked by kubectl exec plugin

----------------------------------------
DELIVERABLES
----------------------------------------

Claude must provide:

1. Architecture design
2. Token flow diagrams for Deploy Step and Test Step
3. Go implementation skeleton
4. Example kubeconfig exec configuration
5. Security model explanation
6. Edge case handling strategy
7. Testing strategy

----------------------------------------
NON-GOALS
----------------------------------------

Do NOT rely on:

- gcloud CLI
- aws CLI
- azure CLI
