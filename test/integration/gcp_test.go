//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider/gcp"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
)

// TestGCPIntegration tests end-to-end GCP token generation and cluster info retrieval
// Requires:
//   - GOOGLE_APPLICATION_CREDENTIALS or --credentials-file pointing to a valid GCP service account
//   - GCP_TEST_PROJECT_ID: GCP project ID
//   - GCP_TEST_CLUSTER_NAME: GKE cluster name
//   - GCP_TEST_REGION: GKE cluster location (region or zone)
func TestGCPIntegration(t *testing.T) {
	// Check required environment variables
	projectID := os.Getenv("GCP_TEST_PROJECT_ID")
	clusterName := os.Getenv("GCP_TEST_CLUSTER_NAME")
	region := os.Getenv("GCP_TEST_REGION")
	credentialsFile := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")

	if projectID == "" || clusterName == "" || region == "" {
		t.Skip("Skipping GCP integration test: missing required environment variables (GCP_TEST_PROJECT_ID, GCP_TEST_CLUSTER_NAME, GCP_TEST_REGION)")
	}

	if credentialsFile == "" {
		t.Skip("Skipping GCP integration test: GOOGLE_APPLICATION_CREDENTIALS not set")
	}

	// Create logger
	log, err := logger.New(logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.ConsoleFormat,
		Output: os.Stderr,
	})
	require.NoError(t, err, "Failed to create logger")

	ctx := context.Background()

	t.Run("CreateProvider", func(t *testing.T) {
		config := &gcp.Config{
			ProjectID:       projectID,
			CredentialsFile: credentialsFile,
			TokenDuration:   1 * time.Hour,
			Scopes:          gcp.DefaultScopes(),
		}

		provider, err := gcp.NewProvider(config, log)
		require.NoError(t, err, "Failed to create GCP provider")
		assert.NotNil(t, provider)
	})

	t.Run("GetToken", func(t *testing.T) {
		config := &gcp.Config{
			ProjectID:       projectID,
			CredentialsFile: credentialsFile,
			TokenDuration:   1 * time.Hour,
			Scopes:          gcp.DefaultScopes(),
		}

		p, err := gcp.NewProvider(config, log)
		require.NoError(t, err, "Failed to create GCP provider")

		opts := provider.GetTokenOptions{
			ClusterName: clusterName,
			Region:      region,
			ProjectID:   projectID,
		}

		token, err := p.GetToken(ctx, opts)
		require.NoError(t, err, "Failed to get token")
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken, "Token should not be empty")
		assert.Equal(t, "Bearer", token.TokenType, "Token type should be Bearer")
		assert.True(t, token.ExpiresAt.After(time.Now()), "Token should not be expired")
		assert.True(t, token.ExpiresAt.Before(time.Now().Add(2*time.Hour)), "Token expiration should be reasonable")

		t.Logf("Token generated successfully, expires at: %s", token.ExpiresAt.Format(time.RFC3339))
	})

	t.Run("GetClusterInfo", func(t *testing.T) {
		config := &gcp.Config{
			ProjectID:       projectID,
			CredentialsFile: credentialsFile,
			TokenDuration:   1 * time.Hour,
			Scopes:          gcp.DefaultScopes(),
		}

		p, err := gcp.NewProvider(config, log)
		require.NoError(t, err, "Failed to create GCP provider")

		info, err := p.GetClusterInfo(ctx, clusterName, region)
		require.NoError(t, err, "Failed to get cluster info")
		assert.NotNil(t, info)
		assert.NotEmpty(t, info.Endpoint, "Endpoint should not be empty")
		assert.NotEmpty(t, info.CertificateAuthority, "CA certificate should not be empty")
		assert.NotEmpty(t, info.Version, "Version should not be empty")
		assert.Equal(t, region, info.Location, "Location should match")

		t.Logf("Cluster info retrieved: endpoint=%s, version=%s, location=%s",
			info.Endpoint, info.Version, info.Location)
	})

	t.Run("ValidateCredentials", func(t *testing.T) {
		config := &gcp.Config{
			ProjectID:       projectID,
			CredentialsFile: credentialsFile,
			TokenDuration:   1 * time.Hour,
			Scopes:          gcp.DefaultScopes(),
		}

		p, err := gcp.NewProvider(config, log)
		require.NoError(t, err, "Failed to create GCP provider")

		err = p.ValidateCredentials(ctx)
		assert.NoError(t, err, "Credentials should be valid")
	})

	t.Run("EndToEnd", func(t *testing.T) {
		// This test simulates the complete workflow:
		// 1. Create provider with credentials
		// 2. Get cluster info
		// 3. Generate token
		// Both should succeed with the same credentials

		config := &gcp.Config{
			ProjectID:       projectID,
			CredentialsFile: credentialsFile,
			TokenDuration:   1 * time.Hour,
			Scopes:          gcp.DefaultScopes(),
		}

		p, err := gcp.NewProvider(config, log)
		require.NoError(t, err, "Failed to create GCP provider")

		// Step 1: Get cluster info (like generate-kubeconfig does)
		clusterInfo, err := p.GetClusterInfo(ctx, clusterName, region)
		require.NoError(t, err, "Failed to get cluster info")
		t.Logf("Step 1: Got cluster info - endpoint: %s", clusterInfo.Endpoint)

		// Step 2: Generate token (like kubectl exec plugin does)
		opts := provider.GetTokenOptions{
			ClusterName: clusterName,
			Region:      region,
			ProjectID:   projectID,
		}

		token, err := p.GetToken(ctx, opts)
		require.NoError(t, err, "Failed to get token")
		t.Logf("Step 2: Got token - expires: %s", token.ExpiresAt.Format(time.RFC3339))

		// Both should work with the same credentials
		assert.NotEmpty(t, clusterInfo.Endpoint)
		assert.NotEmpty(t, token.AccessToken)
	})
}

// TestGCPInvalidCredentials tests that the provider properly handles invalid credentials
func TestGCPInvalidCredentials(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  logger.ErrorLevel,
		Format: logger.ConsoleFormat,
		Output: os.Stderr,
	})
	require.NoError(t, err)

	t.Run("InvalidCredentialsFile", func(t *testing.T) {
		config := &gcp.Config{
			ProjectID:       "fake-project",
			CredentialsFile: "/tmp/nonexistent-credentials.json",
			TokenDuration:   1 * time.Hour,
			Scopes:          gcp.DefaultScopes(),
		}

		p, err := gcp.NewProvider(config, log)
		require.NoError(t, err, "Provider creation should not fail immediately")

		// But using it should fail
		ctx := context.Background()
		_, err = p.GetToken(ctx, provider.GetTokenOptions{
			ClusterName: "test",
			Region:      "us-central1",
			ProjectID:   "fake-project",
		})
		assert.Error(t, err, "Should fail when trying to use invalid credentials")
	})

	t.Run("MissingProjectID", func(t *testing.T) {
		config := &gcp.Config{
			ProjectID:     "", // Missing
			TokenDuration: 1 * time.Hour,
			Scopes:        gcp.DefaultScopes(),
		}

		_, err := gcp.NewProvider(config, log)
		assert.Error(t, err, "Should fail with missing project ID")
	})
}
