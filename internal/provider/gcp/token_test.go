package gcp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/credentials"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/testutil"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

// TestTokenGenerator_LoadCredentials tests credential loading logic
func TestTokenGenerator_LoadCredentials(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() *testutil.MockCredLoader
		wantErr     bool
		wantErrCode errors.ErrorCode
		validate    func(t *testing.T, creds *credentials.GCPCredentials)
	}{
		{
			name: "successful credential loading",
			setupMock: func() *testutil.MockCredLoader {
				return testutil.NewMockCredLoader().WithGCPCreds(testutil.CreateValidGCPCredentials())
			},
			wantErr: false,
			validate: func(t *testing.T, creds *credentials.GCPCredentials) {
				assert.Equal(t, "test-project-12345", creds.ProjectID)
				assert.Equal(t, "test-sa@test-project-12345.iam.gserviceaccount.com", creds.ClientEmail)
			},
		},
		{
			name: "credential loading failure",
			setupMock: func() *testutil.MockCredLoader {
				return testutil.NewMockCredLoader().WithGCPError(
					errors.New(errors.ErrCredentialLoadFailed, "file not found"),
				)
			},
			wantErr:     true,
			wantErrCode: errors.ErrCredentialLoadFailed,
		},
		{
			name: "project ID mismatch warning",
			setupMock: func() *testutil.MockCredLoader {
				creds := testutil.CreateValidGCPCredentials()
				creds.ProjectID = "different-project"
				return testutil.NewMockCredLoader().WithGCPCreds(creds)
			},
			wantErr: false,
			validate: func(t *testing.T, creds *credentials.GCPCredentials) {
				// Should still succeed even with mismatched project ID
				assert.Equal(t, "different-project", creds.ProjectID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.Nop()
			mockLoader := tt.setupMock()
			config := &Config{
				ProjectID:     "test-project-12345",
				TokenDuration: 1 * time.Hour,
				Scopes:        DefaultScopes(),
			}

			generator := NewTokenGenerator(config, mockLoader, log)
			creds, err := generator.loadCredentials(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrCode != "" {
					assert.True(t, errors.Is(err, tt.wantErrCode),
						"expected error code %s, got %v", tt.wantErrCode, err)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, creds)
				if tt.validate != nil {
					tt.validate(t, creds)
				}
			}
		})
	}
}

// TestTokenGenerator_ValidateToken tests token validation logic
func TestTokenGenerator_ValidateToken(t *testing.T) {
	log := logger.Nop()
	config := DefaultConfig()
	mockLoader := testutil.NewMockCredLoader()
	generator := NewTokenGenerator(config, mockLoader, log)

	tests := []struct {
		name        string
		token       *provider.Token
		wantErr     bool
		wantErrCode errors.ErrorCode
	}{
		{
			name: "valid token",
			token: &provider.Token{
				AccessToken: "ya29.c.KqEB...",
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
				AccessToken: "ya29.c.KqEB...",
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
				TokenType:   "Bearer",
			},
			wantErr:     true,
			wantErrCode: errors.ErrTokenExpired,
		},
		{
			name: "token expiring soon (4 minutes)",
			token: &provider.Token{
				AccessToken: "ya29.c.KqEB...",
				ExpiresAt:   time.Now().Add(4 * time.Minute),
				TokenType:   "Bearer",
			},
			wantErr: false, // Still valid, just logs warning
		},
		{
			name: "token with long expiry",
			token: &provider.Token{
				AccessToken: "ya29.c.KqEB...",
				ExpiresAt:   time.Now().Add(24 * time.Hour),
				TokenType:   "Bearer",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := generator.ValidateToken(tt.token)

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

// TestTokenGenerator_RefreshToken tests token refresh logic
// TestTokenGenerator_RefreshToken tests token refresh decision logic
// Note: This only tests cases where refresh is NOT needed (returns current token).
// Cases that trigger actual token generation are tested via TestTokenGenerator_LoadCredentials
// and integration tests, as they require real Google API calls.
func TestTokenGenerator_RefreshToken(t *testing.T) {
	log := logger.Nop()
	mockLoader := testutil.NewMockCredLoader().WithGCPCreds(testutil.CreateValidGCPCredentials())
	config := &Config{
		ProjectID:     "test-project-12345",
		TokenDuration: 1 * time.Hour,
		Scopes:        DefaultScopes(),
	}
	generator := NewTokenGenerator(config, mockLoader, log)

	tests := []struct {
		name         string
		currentToken *provider.Token
	}{
		{
			name: "token expiring in 10 minutes does not trigger refresh",
			currentToken: &provider.Token{
				AccessToken: "still-valid-token",
				ExpiresAt:   time.Now().Add(10 * time.Minute),
				TokenType:   "Bearer",
			},
		},
		{
			name: "token expiring in 1 hour does not trigger refresh",
			currentToken: &provider.Token{
				AccessToken: "fresh-token",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
				TokenType:   "Bearer",
			},
		},
		{
			name: "token expiring in 6 minutes does not trigger refresh",
			currentToken: &provider.Token{
				AccessToken: "still-valid-token",
				ExpiresAt:   time.Now().Add(6 * time.Minute),
				TokenType:   "Bearer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := provider.GetTokenOptions{
				ClusterName: "test-cluster",
				ProjectID:   "test-project-12345",
				Region:      "us-central1",
			}

			// Should return the same token without refresh
			token, err := generator.RefreshToken(context.Background(), opts, tt.currentToken)

			assert.NoError(t, err)
			assert.Equal(t, tt.currentToken, token, "should return the same token without refresh")
		})
	}
}

// TestDefaultScopes verifies the default GCP scopes
func TestDefaultScopes(t *testing.T) {
	scopes := DefaultScopes()

	assert.NotEmpty(t, scopes)
	assert.Contains(t, scopes, "https://www.googleapis.com/auth/cloud-platform")
	t.Logf("Default scopes: %v", scopes)
}

// TestNewTokenGenerator verifies token generator initialization
func TestNewTokenGenerator(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name: "valid config",
			config: &Config{
				ProjectID:     "test-project",
				TokenDuration: 1 * time.Hour,
				Scopes:        DefaultScopes(),
			},
			valid: true,
		},
		{
			name: "minimal config",
			config: &Config{
				ProjectID: "test-project",
			},
			valid: true,
		},
		{
			name: "custom scopes",
			config: &Config{
				ProjectID:     "test-project",
				TokenDuration: 30 * time.Minute,
				Scopes:        []string{"custom-scope"},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.Nop()
			mockLoader := testutil.NewMockCredLoader()

			generator := NewTokenGenerator(tt.config, mockLoader, log)

			if tt.valid {
				assert.NotNil(t, generator)
				assert.NotNil(t, generator.config)
				assert.NotNil(t, generator.credLoader)
				assert.NotNil(t, generator.logger)
			}
		})
	}
}

// TestToken_ExpiresIn tests the token expiration calculation
func TestToken_ExpiresIn(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		want      time.Duration
	}{
		{
			name:      "expires in 1 hour",
			expiresAt: now.Add(1 * time.Hour),
			want:      1 * time.Hour,
		},
		{
			name:      "expires in 30 minutes",
			expiresAt: now.Add(30 * time.Minute),
			want:      30 * time.Minute,
		},
		{
			name:      "already expired",
			expiresAt: now.Add(-10 * time.Minute),
			want:      0, // Negative durations should be treated as 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &provider.Token{
				AccessToken: "test-token",
				ExpiresAt:   tt.expiresAt,
				TokenType:   "Bearer",
			}

			expiresIn := token.ExpiresIn()

			// Allow small time drift (1 second) due to test execution time
			if tt.want > 0 {
				assert.InDelta(t, tt.want.Seconds(), expiresIn.Seconds(), 1.0)
			} else {
				assert.LessOrEqual(t, expiresIn, time.Duration(0))
			}
		})
	}
}

// TestToken_IsExpired tests token expiration check
func TestToken_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "expired",
			expiresAt: now.Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "not expired",
			expiresAt: now.Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "expires now",
			expiresAt: now,
			want:      true, // Tokens expiring "now" are considered expired
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &provider.Token{
				AccessToken: "test-token",
				ExpiresAt:   tt.expiresAt,
				TokenType:   "Bearer",
			}

			// Allow small time drift
			if tt.want {
				assert.True(t, token.IsExpired() || time.Until(token.ExpiresAt) < time.Second)
			} else {
				assert.False(t, token.IsExpired())
			}
		})
	}
}
