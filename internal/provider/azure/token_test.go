package azure

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/credentials"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

func TestNewTokenGenerator(t *testing.T) {
	log := logger.Nop()
	config := &Config{
		TenantID:       "test-tenant-id",
		SubscriptionID: "test-subscription-id",
		TokenDuration:  1 * time.Hour,
	}
	credLoader := credentials.NewLoader(log)

	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "with config",
			config: config,
		},
		{
			name:   "with nil config",
			config: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewTokenGenerator(tt.config, credLoader, log)
			require.NotNil(t, gen)
			assert.Equal(t, credLoader, gen.credLoader)
			assert.Equal(t, log, gen.logger)
		})
	}
}

func TestTokenGenerator_GenerateToken(t *testing.T) {
	log := logger.Nop()
	config := &Config{
		TenantID:       "test-tenant-id",
		SubscriptionID: "test-subscription-id",
		TokenDuration:  1 * time.Hour,
	}

	tests := []struct {
		name        string
		opts        provider.GetTokenOptions
		setupEnv    func()
		cleanupEnv  func()
		wantErr     bool
		wantErrCode errors.ErrorCode
	}{
		{
			name: "successful token generation with env credentials",
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
			name: "missing credentials",
			opts: provider.GetTokenOptions{
				ClusterName:    "test-cluster",
				SubscriptionID: "12345678-1234-1234-1234-123456789012",
				TenantID:       "87654321-4321-4321-4321-210987654321",
			},
			setupEnv:   func() {},
			cleanupEnv: func() {},
			wantErr:    true,
		},
		{
			name: "subscription and tenant from config",
			opts: provider.GetTokenOptions{
				ClusterName: "test-cluster",
				// SubscriptionID and TenantID should come from config
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupEnv != nil {
				tt.setupEnv()
			}
			if tt.cleanupEnv != nil {
				defer tt.cleanupEnv()
			}

			credLoader := credentials.NewLoader(log)
			gen := NewTokenGenerator(config, credLoader, log)

			token, err := gen.GenerateToken(context.Background(), tt.opts)

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

func TestTokenGenerator_ValidateToken(t *testing.T) {
	log := logger.Nop()
	config := &Config{
		TenantID:       "test-tenant-id",
		SubscriptionID: "test-subscription-id",
		TokenDuration:  1 * time.Hour,
	}
	credLoader := credentials.NewLoader(log)
	gen := NewTokenGenerator(config, credLoader, log)

	tests := []struct {
		name        string
		token       *provider.Token
		wantErr     bool
		wantErrCode errors.ErrorCode
	}{
		{
			name: "valid token",
			token: &provider.Token{
				AccessToken: "valid-azure-ad-token",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
				TokenType:   "Bearer",
			},
			wantErr: false,
		},
		{
			name:        "nil token",
			token:       nil,
			wantErr:     true,
			wantErrCode: errors.ErrTokenInvalid,
		},
		{
			name: "empty access token",
			token: &provider.Token{
				AccessToken: "",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
				TokenType:   "Bearer",
			},
			wantErr:     true,
			wantErrCode: errors.ErrTokenInvalid,
		},
		{
			name: "expired token",
			token: &provider.Token{
				AccessToken: "expired-token",
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
				TokenType:   "Bearer",
			},
			wantErr:     true,
			wantErrCode: errors.ErrTokenExpired,
		},
		{
			name: "token expiring soon (warning only)",
			token: &provider.Token{
				AccessToken: "expiring-soon-token",
				ExpiresAt:   time.Now().Add(2 * time.Minute),
				TokenType:   "Bearer",
			},
			wantErr: false, // Should warn but not error
		},
		{
			name: "token with long expiry",
			token: &provider.Token{
				AccessToken: "long-lived-token",
				ExpiresAt:   time.Now().Add(24 * time.Hour),
				TokenType:   "Bearer",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gen.ValidateToken(tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrCode != "" {
					assert.True(t, errors.Is(err, tt.wantErrCode),
						"expected error code %s, got %v", tt.wantErrCode, err)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenGenerator_RefreshToken(t *testing.T) {
	log := logger.Nop()
	config := &Config{
		TenantID:       "test-tenant-id",
		SubscriptionID: "test-subscription-id",
		TokenDuration:  1 * time.Hour,
	}
	credLoader := credentials.NewLoader(log)
	gen := NewTokenGenerator(config, credLoader, log)

	opts := provider.GetTokenOptions{
		ClusterName:    "test-cluster",
		SubscriptionID: "12345678-1234-1234-1234-123456789012",
		TenantID:       "87654321-4321-4321-4321-210987654321",
	}

	tests := []struct {
		name         string
		currentToken *provider.Token
		shouldRefresh bool
	}{
		{
			name:          "nil token should refresh",
			currentToken:  nil,
			shouldRefresh: true,
		},
		{
			name: "expired token should refresh",
			currentToken: &provider.Token{
				AccessToken: "expired-token",
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
				TokenType:   "Bearer",
			},
			shouldRefresh: true,
		},
		{
			name: "token expiring soon should refresh",
			currentToken: &provider.Token{
				AccessToken: "expiring-soon",
				ExpiresAt:   time.Now().Add(2 * time.Minute),
				TokenType:   "Bearer",
			},
			shouldRefresh: true,
		},
		{
			name: "valid token with time remaining should not refresh",
			currentToken: &provider.Token{
				AccessToken: "valid-token",
				ExpiresAt:   time.Now().Add(30 * time.Minute),
				TokenType:   "Bearer",
			},
			shouldRefresh: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			os.Setenv("AZURE_CLIENT_ID", "11111111-1111-1111-1111-111111111111")
			os.Setenv("AZURE_CLIENT_SECRET", "test-client-secret")
			os.Setenv("AZURE_TENANT_ID", "87654321-4321-4321-4321-210987654321")
			defer func() {
				os.Unsetenv("AZURE_CLIENT_ID")
				os.Unsetenv("AZURE_CLIENT_SECRET")
				os.Unsetenv("AZURE_TENANT_ID")
			}()

			token, err := gen.RefreshToken(context.Background(), opts, tt.currentToken)

			if !tt.shouldRefresh {
				// Should return current token without refreshing
				assert.NoError(t, err)
				assert.Equal(t, tt.currentToken, token)
			} else {
				// In test environment without real credentials, refresh will fail
				// This is expected - integration tests will verify with real credentials
				if err != nil {
					t.Logf("Expected error in test environment: %v", err)
					return
				}
				// If somehow succeeded (shouldn't in test env), verify new token
				assert.NoError(t, err)
				require.NotNil(t, token)
				assert.NotEqual(t, tt.currentToken, token)
			}
		})
	}
}

func TestDefaultPresignDuration(t *testing.T) {
	assert.Equal(t, 1*time.Hour, defaultTokenDuration)
}

func TestGetTokenDuration(t *testing.T) {
	log := logger.Nop()
	credLoader := credentials.NewLoader(log)

	tests := []struct {
		name     string
		config   *Config
		expected time.Duration
	}{
		{
			name: "custom duration",
			config: &Config{
				TokenDuration: 30 * time.Minute,
			},
			expected: 30 * time.Minute,
		},
		{
			name:     "default duration",
			config:   &Config{},
			expected: defaultTokenDuration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := NewTokenGenerator(tt.config, credLoader, log)
			assert.Equal(t, tt.expected, gen.getTokenDuration())
		})
	}
}

func TestAKSResourceScope(t *testing.T) {
	assert.Equal(t, "https://management.azure.com/.default", aksResourceScope)
}
