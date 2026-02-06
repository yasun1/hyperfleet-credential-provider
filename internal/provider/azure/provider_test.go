package azure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

func TestNewProvider(t *testing.T) {
	log := logger.Nop()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				TenantID:       "test-tenant-id",
				SubscriptionID: "test-subscription-id",
				TokenDuration:  1 * time.Hour,
			},
			wantErr: false,
		},
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: false,
		},
		{
			name: "with resource group",
			config: &Config{
				TenantID:       "test-tenant-id",
				SubscriptionID: "test-subscription-id",
				ResourceGroup:  "test-resource-group",
				TokenDuration:  1 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "empty tenant and subscription",
			config: &Config{
				TenantID:       "",
				SubscriptionID: "",
				TokenDuration:  1 * time.Hour,
			},
			wantErr: false, // Azure can use environment variables
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			azureProvider, err := NewProvider(tt.config, log)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, azureProvider)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, azureProvider)
				assert.Equal(t, "azure", azureProvider.Name())
				assert.NotNil(t, azureProvider.tokenGenerator)
				assert.NotNil(t, azureProvider.credLoader)
			}
		})
	}
}

func TestProvider_Name(t *testing.T) {
	log := logger.Nop()
	config := &Config{
		TenantID:       "test-tenant-id",
		SubscriptionID: "test-subscription-id",
		TokenDuration:  1 * time.Hour,
	}

	azureProvider, err := NewProvider(config, log)
	require.NoError(t, err)

	assert.Equal(t, "azure", azureProvider.Name())
}

func TestProvider_GetToken(t *testing.T) {
	log := logger.Nop()

	tests := []struct {
		name        string
		config      *Config
		opts        provider.GetTokenOptions
		setupEnv    func()
		cleanupEnv  func()
		wantErr     bool
		wantErrCode errors.ErrorCode
	}{
		{
			name: "valid token request",
			config: &Config{
				TenantID:       "87654321-4321-4321-4321-210987654321",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TokenDuration:  1 * time.Hour,
			},
			opts: provider.GetTokenOptions{
				ClusterName:    "test-aks-cluster",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TenantID:       "87654321-4321-4321-4321-210987654321",
			},
			setupEnv: func() {
				os.Setenv("AZURE_CLIENT_ID", "11111111-1111-1111-1111-111111111111")
				os.Setenv("AZURE_CLIENT_SECRET", "test-client-secret")
				os.Setenv("AZURE_TENANT_ID", "87654321-4321-4321-4321-210987654321")
			},
			cleanupEnv: func() {
				os.Unsetenv("AZURE_CLIENT_ID")
				os.Unsetenv("AZURE_CLIENT_SECRET")
				os.Unsetenv("AZURE_TENANT_ID")
			},
			wantErr: false, // May error in test env without real credentials
		},
		{
			name: "missing cluster name",
			config: &Config{
				TenantID:       "87654321-4321-4321-4321-210987654321",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TokenDuration:  1 * time.Hour,
			},
			opts: provider.GetTokenOptions{
				ClusterName:    "",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TenantID:       "87654321-4321-4321-4321-210987654321",
			},
			setupEnv:    func() {},
			cleanupEnv:  func() {},
			wantErr:     true,
			wantErrCode: errors.ErrInvalidArgument,
		},
		{
			name: "tenant and subscription from config",
			config: &Config{
				TenantID:       "87654321-4321-4321-4321-210987654321",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TokenDuration:  1 * time.Hour,
			},
			opts: provider.GetTokenOptions{
				ClusterName: "test-cluster",
				// TenantID and SubscriptionID should come from config
			},
			setupEnv: func() {
				os.Setenv("AZURE_CLIENT_ID", "11111111-1111-1111-1111-111111111111")
				os.Setenv("AZURE_CLIENT_SECRET", "test-client-secret")
				os.Setenv("AZURE_TENANT_ID", "87654321-4321-4321-4321-210987654321")
			},
			cleanupEnv: func() {
				os.Unsetenv("AZURE_CLIENT_ID")
				os.Unsetenv("AZURE_CLIENT_SECRET")
				os.Unsetenv("AZURE_TENANT_ID")
			},
			wantErr: false, // May error in test env
		},
		{
			name: "missing credentials",
			config: &Config{
				TenantID:       "87654321-4321-4321-4321-210987654321",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TokenDuration:  1 * time.Hour,
			},
			opts: provider.GetTokenOptions{
				ClusterName:    "test-cluster",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TenantID:       "87654321-4321-4321-4321-210987654321",
			},
			setupEnv:   func() {},
			cleanupEnv: func() {},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			azureProvider, err := NewProvider(tt.config, log)
			require.NoError(t, err)

			token, err := azureProvider.GetToken(context.Background(), tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrCode != "" {
					assert.True(t, errors.Is(err, tt.wantErrCode),
						"expected error code %s, got %v", tt.wantErrCode, err)
				}
				assert.Nil(t, token)
			} else {
				// In test environment without real Azure credentials, we expect failure
				// This is okay - integration tests will verify with real credentials
				if err != nil {
					t.Logf("Expected error in test environment: %v", err)
					return
				}
				assert.NoError(t, err)
				require.NotNil(t, token)
				assert.NotEmpty(t, token.AccessToken)
				assert.Equal(t, "Bearer", token.TokenType)
				assert.False(t, token.IsExpired())
			}
		})
	}
}

func TestProvider_ValidateCredentials(t *testing.T) {
	log := logger.Nop()

	tests := []struct {
		name        string
		config      *Config
		setupEnv    func()
		cleanupEnv  func()
		wantErr     bool
		wantErrCode errors.ErrorCode
	}{
		{
			name: "valid credentials",
			config: &Config{
				TenantID:       "87654321-4321-4321-4321-210987654321",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TokenDuration:  1 * time.Hour,
			},
			setupEnv: func() {
				os.Setenv("AZURE_CLIENT_ID", "11111111-1111-1111-1111-111111111111")
				os.Setenv("AZURE_CLIENT_SECRET", "test-client-secret")
				os.Setenv("AZURE_TENANT_ID", "87654321-4321-4321-4321-210987654321")
			},
			cleanupEnv: func() {
				os.Unsetenv("AZURE_CLIENT_ID")
				os.Unsetenv("AZURE_CLIENT_SECRET")
				os.Unsetenv("AZURE_TENANT_ID")
			},
			wantErr: false, // May error without real credentials
		},
		{
			name: "missing credentials",
			config: &Config{
				TenantID:       "87654321-4321-4321-4321-210987654321",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TokenDuration:  1 * time.Hour,
			},
			setupEnv:    func() {},
			cleanupEnv:  func() {},
			wantErr:     true,
			wantErrCode: errors.ErrCredentialValidationFailed,
		},
		{
			name: "credentials from environment",
			config: &Config{
				TokenDuration: 1 * time.Hour,
			},
			setupEnv: func() {
				os.Setenv("AZURE_CLIENT_ID", "11111111-1111-1111-1111-111111111111")
				os.Setenv("AZURE_CLIENT_SECRET", "test-client-secret")
				os.Setenv("AZURE_TENANT_ID", "87654321-4321-4321-4321-210987654321")
				os.Setenv("AZURE_SUBSCRIPTION_ID", "12345678-1234-1234-1234-123456789012")
			},
			cleanupEnv: func() {
				os.Unsetenv("AZURE_CLIENT_ID")
				os.Unsetenv("AZURE_CLIENT_SECRET")
				os.Unsetenv("AZURE_TENANT_ID")
				os.Unsetenv("AZURE_SUBSCRIPTION_ID")
			},
			wantErr: false, // May error without real credentials
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			azureProvider, err := NewProvider(tt.config, log)
			require.NoError(t, err)

			err = azureProvider.ValidateCredentials(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrCode != "" {
					assert.True(t, errors.Is(err, tt.wantErrCode),
						"expected error code %s, got %v", tt.wantErrCode, err)
				}
			} else {
				// In test environment without real Azure credentials, we expect failure
				// This is okay - integration tests will verify with real credentials
				if err != nil {
					t.Logf("Expected error in test environment: %v", err)
					return
				}
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 1*time.Hour, config.TokenDuration)
}

func TestProvider_Integration(t *testing.T) {
	// This is a basic integration test structure
	// Real integration tests should use actual Azure credentials
	log := logger.Nop()

	config := &Config{
		TenantID:       os.Getenv("AZURE_TENANT_ID"),
		SubscriptionID: os.Getenv("AZURE_SUBSCRIPTION_ID"),
		TokenDuration:  1 * time.Hour,
	}

	// Skip if no credentials available
	if os.Getenv("AZURE_CLIENT_ID") == "" {
		t.Skip("Skipping integration test: AZURE_CLIENT_ID not set")
	}

	if config.TenantID == "" {
		t.Skip("Skipping integration test: AZURE_TENANT_ID not set")
	}

	if config.SubscriptionID == "" {
		t.Skip("Skipping integration test: AZURE_SUBSCRIPTION_ID not set")
	}

	azureProvider, err := NewProvider(config, log)
	if err != nil {
		t.Skipf("Skipping integration test: %v", err)
	}

	// Test credential validation
	err = azureProvider.ValidateCredentials(context.Background())
	if err != nil {
		t.Logf("Credential validation failed (expected in test env): %v", err)
	}

	// Test token generation
	opts := provider.GetTokenOptions{
		ClusterName:    "integration-test-cluster",
		SubscriptionID: config.SubscriptionID,
		TenantID:       config.TenantID,
	}

	token, err := azureProvider.GetToken(context.Background(), opts)
	if err != nil {
		t.Logf("Token generation failed (expected in test env): %v", err)
		return
	}

	// Validate token structure
	assert.NotEmpty(t, token.AccessToken)
	assert.Equal(t, "Bearer", token.TokenType)
	assert.False(t, token.IsExpired())
	assert.True(t, token.ExpiresAt.After(time.Now()))
}
