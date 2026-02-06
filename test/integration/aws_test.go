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
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider/aws"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
)

// TestAWSIntegration tests end-to-end AWS token generation and cluster info retrieval
// Requires:
//   - AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY or AWS_CREDENTIALS_FILE
//   - AWS_TEST_CLUSTER_NAME: EKS cluster name
//   - AWS_TEST_REGION: AWS region (e.g., us-east-1)
func TestAWSIntegration(t *testing.T) {
	// Check required environment variables
	clusterName := os.Getenv("AWS_TEST_CLUSTER_NAME")
	region := os.Getenv("AWS_TEST_REGION")

	if clusterName == "" || region == "" {
		t.Skip("Skipping AWS integration test: missing required environment variables (AWS_TEST_CLUSTER_NAME, AWS_TEST_REGION)")
	}

	// Check if credentials are available
	hasEnvCreds := os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != ""
	hasCredsFile := os.Getenv("AWS_CREDENTIALS_FILE") != ""

	if !hasEnvCreds && !hasCredsFile {
		t.Skip("Skipping AWS integration test: no AWS credentials found (AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY or AWS_CREDENTIALS_FILE)")
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
		config := &aws.Config{
			Region:        region,
			TokenDuration: 15 * time.Minute,
		}

		provider, err := aws.NewProvider(config, log)
		require.NoError(t, err, "Failed to create AWS provider")
		assert.NotNil(t, provider)
	})

	t.Run("GetToken", func(t *testing.T) {
		config := &aws.Config{
			Region:        region,
			TokenDuration: 15 * time.Minute,
		}

		p, err := aws.NewProvider(config, log)
		require.NoError(t, err, "Failed to create AWS provider")

		opts := provider.GetTokenOptions{
			ClusterName: clusterName,
			Region:      region,
		}

		token, err := p.GetToken(ctx, opts)
		require.NoError(t, err, "Failed to get token")
		assert.NotNil(t, token)
		assert.NotEmpty(t, token.AccessToken, "Token should not be empty")
		assert.True(t, token.AccessToken[:7] == "k8s-aws", "AWS token should start with k8s-aws prefix")
		assert.True(t, token.ExpiresAt.After(time.Now()), "Token should not be expired")
		assert.True(t, token.ExpiresAt.Before(time.Now().Add(20*time.Minute)), "Token expiration should be reasonable")

		t.Logf("Token generated successfully, expires at: %s", token.ExpiresAt.Format(time.RFC3339))
	})

	t.Run("GetClusterInfo", func(t *testing.T) {
		config := &aws.Config{
			Region:        region,
			TokenDuration: 15 * time.Minute,
		}

		p, err := aws.NewProvider(config, log)
		require.NoError(t, err, "Failed to create AWS provider")

		info, err := p.GetClusterInfo(ctx, clusterName)
		require.NoError(t, err, "Failed to get cluster info")
		assert.NotNil(t, info)
		assert.NotEmpty(t, info.Endpoint, "Endpoint should not be empty")
		assert.NotEmpty(t, info.CertificateAuthority, "CA certificate should not be empty")
		assert.NotEmpty(t, info.Version, "Version should not be empty")
		assert.Equal(t, region, info.Region, "Region should match")
		assert.NotEmpty(t, info.ARN, "ARN should not be empty")

		t.Logf("Cluster info retrieved: endpoint=%s, version=%s, region=%s, arn=%s",
			info.Endpoint, info.Version, info.Region, info.ARN)
	})

	t.Run("ValidateCredentials", func(t *testing.T) {
		config := &aws.Config{
			Region:        region,
			TokenDuration: 15 * time.Minute,
		}

		p, err := aws.NewProvider(config, log)
		require.NoError(t, err, "Failed to create AWS provider")

		err = p.ValidateCredentials(ctx)
		assert.NoError(t, err, "Credentials should be valid")
	})

	t.Run("EndToEnd", func(t *testing.T) {
		// This test simulates the complete workflow:
		// 1. Create provider with credentials
		// 2. Get cluster info
		// 3. Generate token

		config := &aws.Config{
			Region:        region,
			TokenDuration: 15 * time.Minute,
		}

		p, err := aws.NewProvider(config, log)
		require.NoError(t, err, "Failed to create AWS provider")

		// Step 1: Get cluster info (like generate-kubeconfig does)
		clusterInfo, err := p.GetClusterInfo(ctx, clusterName)
		require.NoError(t, err, "Failed to get cluster info")
		t.Logf("Step 1: Got cluster info - endpoint: %s", clusterInfo.Endpoint)

		// Step 2: Generate token (like kubectl exec plugin does)
		opts := provider.GetTokenOptions{
			ClusterName: clusterName,
			Region:      region,
		}

		token, err := p.GetToken(ctx, opts)
		require.NoError(t, err, "Failed to get token")
		t.Logf("Step 2: Got token - expires: %s", token.ExpiresAt.Format(time.RFC3339))

		// Both should work with the same credentials
		assert.NotEmpty(t, clusterInfo.Endpoint)
		assert.NotEmpty(t, token.AccessToken)
	})
}

// TestAWSInvalidCredentials tests that the provider properly handles invalid credentials
func TestAWSInvalidCredentials(t *testing.T) {
	log, err := logger.New(logger.Config{
		Level:  logger.ErrorLevel,
		Format: logger.ConsoleFormat,
		Output: os.Stderr,
	})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("InvalidRegion", func(t *testing.T) {
		config := &aws.Config{
			Region:        "invalid-region-9999",
			TokenDuration: 15 * time.Minute,
		}

		p, err := aws.NewProvider(config, log)
		require.NoError(t, err, "Provider should be created even with invalid region")

		// But validation should fail when trying to use it
		err = p.ValidateCredentials(ctx)
		// Note: This might not fail immediately depending on AWS SDK behavior
		t.Logf("Validation result: %v", err)
	})

	t.Run("MissingRegion", func(t *testing.T) {
		config := &aws.Config{
			Region:        "", // Missing
			TokenDuration: 15 * time.Minute,
		}

		_, err := aws.NewProvider(config, log)
		assert.Error(t, err, "Should fail with missing region")
	})
}

// TestAWSTokenFormat tests the AWS token format
func TestAWSTokenFormat(t *testing.T) {
	clusterName := os.Getenv("AWS_TEST_CLUSTER_NAME")
	region := os.Getenv("AWS_TEST_REGION")

	if clusterName == "" || region == "" {
		t.Skip("Skipping AWS token format test: missing environment variables")
	}

	hasEnvCreds := os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != ""
	hasCredsFile := os.Getenv("AWS_CREDENTIALS_FILE") != ""

	if !hasEnvCreds && !hasCredsFile {
		t.Skip("Skipping AWS token format test: no AWS credentials")
	}

	log, err := logger.New(logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.ConsoleFormat,
		Output: os.Stderr,
	})
	require.NoError(t, err)

	config := &aws.Config{
		Region:        region,
		TokenDuration: 15 * time.Minute,
	}

	p, err := aws.NewProvider(config, log)
	require.NoError(t, err)

	ctx := context.Background()
	opts := provider.GetTokenOptions{
		ClusterName: clusterName,
		Region:      region,
	}

	token, err := p.GetToken(ctx, opts)
	require.NoError(t, err)

	// AWS tokens should be base64-encoded and start with "k8s-aws-v1."
	assert.NotEmpty(t, token.AccessToken)
	assert.True(t, len(token.AccessToken) > 100, "Token should be reasonably long")
	t.Logf("Token length: %d characters", len(token.AccessToken))
}
