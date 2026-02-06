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

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/azure"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

// TestAzureIntegration tests end-to-end Azure token generation and cluster info retrieval
// Requires:
//   - AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, AZURE_TENANT_ID or AZURE_CREDENTIALS_FILE
//   - AZURE_TEST_CLUSTER_NAME: AKS cluster name
//   - AZURE_TEST_SUBSCRIPTION_ID: Azure subscription ID
//   - AZURE_TEST_RESOURCE_GROUP: Resource group name
func TestAzureIntegration(t *testing.T) {
	// Check required environment variables
	clusterName := os.Getenv("AZURE_TEST_CLUSTER_NAME")
	subscriptionID := os.Getenv("AZURE_TEST_SUBSCRIPTION_ID")
	resourceGroup := os.Getenv("AZURE_TEST_RESOURCE_GROUP")
	tenantID := os.Getenv("AZURE_TENANT_ID")

	if clusterName == "" || subscriptionID == "" || resourceGroup == "" || tenantID == "" {
		t.Skip("Skipping Azure integration test: missing required environment variables (AZURE_TEST_CLUSTER_NAME, AZURE_TEST_SUBSCRIPTION_ID, AZURE_TEST_RESOURCE_GROUP, AZURE_TENANT_ID)")
	}

	// Check if credentials are available
	hasEnvCreds := os.Getenv("AZURE_CLIENT_ID") != "" && os.Getenv("AZURE_CLIENT_SECRET") != ""
	hasCredsFile := os.Getenv("AZURE_CREDENTIALS_FILE") != ""

	if !hasEnvCreds && !hasCredsFile {
		t.Skip("Skipping Azure integration test: no Azure credentials found (AZURE_CLIENT_ID/AZURE_CLIENT_SECRET or AZURE_CREDENTIALS_FILE)")
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
		config := &azure.Config{
			TenantID:       tenantID,
			SubscriptionID: subscriptionID,
			TokenDuration:  1 * time.Hour,
		}

		provider, err := azure.NewProvider(config, log)
		require.NoError(t, err, "Failed to create Azure provider")
		assert.NotNil(t, provider)
	})

	t.Run("GetToken", func(t *testing.T) {
		config := &azure.Config{
			TenantID:       tenantID,
			SubscriptionID: subscriptionID,
			TokenDuration:  1 * time.Hour,
		}

		p, err := azure.NewProvider(config, log)
		require.NoError(t, err, "Failed to create Azure provider")

		opts := provider.GetTokenOptions{
			ClusterName:    clusterName,
			SubscriptionID: subscriptionID,
			TenantID:       tenantID,
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
		config := &azure.Config{
			TenantID:       tenantID,
			SubscriptionID: subscriptionID,
			TokenDuration:  1 * time.Hour,
		}

		p, err := azure.NewProvider(config, log)
		require.NoError(t, err, "Failed to create Azure provider")

		info, err := p.GetClusterInfo(ctx, clusterName, resourceGroup)
		require.NoError(t, err, "Failed to get cluster info")
		assert.NotNil(t, info)
		assert.NotEmpty(t, info.Endpoint, "Endpoint should not be empty")
		assert.NotEmpty(t, info.CertificateAuthority, "CA certificate should not be empty")
		assert.NotEmpty(t, info.Version, "Version should not be empty")
		assert.NotEmpty(t, info.Location, "Location should not be empty")
		assert.NotEmpty(t, info.ResourceID, "Resource ID should not be empty")

		t.Logf("Cluster info retrieved: endpoint=%s, version=%s, location=%s",
			info.Endpoint, info.Version, info.Location)
	})

	t.Run("ValidateCredentials", func(t *testing.T) {
		config := &azure.Config{
			TenantID:       tenantID,
			SubscriptionID: subscriptionID,
			TokenDuration:  1 * time.Hour,
		}

		p, err := azure.NewProvider(config, log)
		require.NoError(t, err, "Failed to create Azure provider")

		err = p.ValidateCredentials(ctx)
		assert.NoError(t, err, "Credentials should be valid")
	})

	t.Run("EndToEnd", func(t *testing.T) {
		// This test simulates the complete workflow:
		// 1. Create provider with credentials
		// 2. Get cluster info
		// 3. Generate token

		config := &azure.Config{
			TenantID:       tenantID,
			SubscriptionID: subscriptionID,
			TokenDuration:  1 * time.Hour,
		}

		p, err := azure.NewProvider(config, log)
		require.NoError(t, err, "Failed to create Azure provider")

		// Step 1: Get cluster info (like generate-kubeconfig does)
		clusterInfo, err := p.GetClusterInfo(ctx, clusterName, resourceGroup)
		require.NoError(t, err, "Failed to get cluster info")
		t.Logf("Step 1: Got cluster info - endpoint: %s", clusterInfo.Endpoint)

		// Step 2: Generate token (like kubectl exec plugin does)
		opts := provider.GetTokenOptions{
			ClusterName:    clusterName,
			SubscriptionID: subscriptionID,
			TenantID:       tenantID,
		}

		token, err := p.GetToken(ctx, opts)
		require.NoError(t, err, "Failed to get token")
		t.Logf("Step 2: Got token - expires: %s", token.ExpiresAt.Format(time.RFC3339))

		// Both should work with the same credentials
		assert.NotEmpty(t, clusterInfo.Endpoint)
		assert.NotEmpty(t, token.AccessToken)
	})
}

// TestAzureInvalidCredentials tests that the provider properly handles invalid credentials
func TestAzureInvalidCredentials(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  logger.ErrorLevel,
		Format: logger.ConsoleFormat,
		Output: os.Stderr,
	})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("InvalidTenantID", func(t *testing.T) {
		config := &azure.Config{
			TenantID:       "00000000-0000-0000-0000-000000000000",
			SubscriptionID: "00000000-0000-0000-0000-000000000000",
			TokenDuration:  1 * time.Hour,
		}

		p, err := azure.NewProvider(config, log)
		require.NoError(t, err, "Provider should be created even with invalid tenant")

		// But validation should fail
		err = p.ValidateCredentials(ctx)
		assert.Error(t, err, "Should fail with invalid tenant ID")
	})

	t.Run("MissingTenantID", func(t *testing.T) {
		config := &azure.Config{
			TenantID:       "", // Missing
			SubscriptionID: "00000000-0000-0000-0000-000000000000",
			TokenDuration:  1 * time.Hour,
		}

		_, err := azure.NewProvider(config, log)
		assert.Error(t, err, "Should fail with missing tenant ID")
	})

	t.Run("MissingSubscriptionID", func(t *testing.T) {
		config := &azure.Config{
			TenantID:       "00000000-0000-0000-0000-000000000000",
			SubscriptionID: "", // Missing
			TokenDuration:  1 * time.Hour,
		}

		_, err := azure.NewProvider(config, log)
		assert.Error(t, err, "Should fail with missing subscription ID")
	})
}

// TestAzureTokenFormat tests the Azure token format
func TestAzureTokenFormat(t *testing.T) {
	clusterName := os.Getenv("AZURE_TEST_CLUSTER_NAME")
	subscriptionID := os.Getenv("AZURE_TEST_SUBSCRIPTION_ID")
	tenantID := os.Getenv("AZURE_TENANT_ID")

	if clusterName == "" || subscriptionID == "" || tenantID == "" {
		t.Skip("Skipping Azure token format test: missing environment variables")
	}

	hasEnvCreds := os.Getenv("AZURE_CLIENT_ID") != "" && os.Getenv("AZURE_CLIENT_SECRET") != ""
	hasCredsFile := os.Getenv("AZURE_CREDENTIALS_FILE") != ""

	if !hasEnvCreds && !hasCredsFile {
		t.Skip("Skipping Azure token format test: no Azure credentials")
	}

	log, err := logger.New(logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.ConsoleFormat,
		Output: os.Stderr,
	})
	require.NoError(t, err)

	config := &azure.Config{
		TenantID:       tenantID,
		SubscriptionID: subscriptionID,
		TokenDuration:  1 * time.Hour,
	}

	p, err := azure.NewProvider(config, log)
	require.NoError(t, err)

	ctx := context.Background()
	opts := provider.GetTokenOptions{
		ClusterName:    clusterName,
		SubscriptionID: subscriptionID,
		TenantID:       tenantID,
	}

	token, err := p.GetToken(ctx, opts)
	require.NoError(t, err)

	// Azure tokens should be JWT format
	assert.NotEmpty(t, token.AccessToken)
	assert.True(t, len(token.AccessToken) > 100, "Token should be reasonably long")
	// JWT tokens typically have 3 parts separated by dots
	// (but we don't want to be too strict here)
	t.Logf("Token length: %d characters", len(token.AccessToken))
}
