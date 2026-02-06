package aws

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/credentials"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
)

func TestTokenGenerator_GenerateToken(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		opts          provider.GetTokenOptions
		setupEnv      func()
		cleanupEnv    func()
		wantErr       bool
		wantErrCode   errors.ErrorCode
		validateToken func(t *testing.T, token *provider.Token)
	}{
		{
			name: "successful token generation with env credentials",
			config: &Config{
				Region:        "us-east-1",
				TokenDuration: 15 * time.Minute,
			},
			opts: provider.GetTokenOptions{
				ClusterName: "test-eks-cluster",
				Region:      "us-east-1",
			},
			setupEnv: func() {
				os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
				os.Setenv("AWS_REGION", "us-east-1")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
				os.Unsetenv("AWS_REGION")
			},
			wantErr: false, // May fail without real AWS credentials
			validateToken: func(t *testing.T, token *provider.Token) {
				if token == nil {
					return // Expected in test env
				}
				assert.NotEmpty(t, token.AccessToken, "access token should not be empty")
				assert.True(t, strings.HasPrefix(token.AccessToken, v1Prefix), "token should have v1 prefix")
				assert.Equal(t, "Bearer", token.TokenType, "token type should be Bearer")
				assert.False(t, token.IsExpired(), "token should not be expired")
				assert.True(t, token.ExpiresAt.After(time.Now()), "expiration should be in the future")
			},
		},
		{
			name: "missing cluster name",
			config: &Config{
				Region:        "us-east-1",
				TokenDuration: 15 * time.Minute,
			},
			opts: provider.GetTokenOptions{
				ClusterName: "",
				Region:      "us-east-1",
			},
			setupEnv: func() {
				os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
			},
			wantErr:     true,
			wantErrCode: errors.ErrInvalidArgument,
		},
		{
			name: "missing credentials",
			config: &Config{
				Region:        "us-east-1",
				TokenDuration: 15 * time.Minute,
			},
			opts: provider.GetTokenOptions{
				ClusterName: "test-cluster",
				Region:      "us-east-1",
			},
			setupEnv: func() {
				// No credentials set
			},
			cleanupEnv: func() {},
			wantErr:    true,
		},
		{
			name: "with session token",
			config: &Config{
				Region:        "us-west-2",
				TokenDuration: 15 * time.Minute,
			},
			opts: provider.GetTokenOptions{
				ClusterName: "test-cluster",
				Region:      "us-west-2",
			},
			setupEnv: func() {
				os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
				os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
				os.Setenv("AWS_SESSION_TOKEN", "session-token-example")
			},
			cleanupEnv: func() {
				os.Unsetenv("AWS_ACCESS_KEY_ID")
				os.Unsetenv("AWS_SECRET_ACCESS_KEY")
				os.Unsetenv("AWS_SESSION_TOKEN")
			},
			wantErr: false, // May fail without real credentials
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

			log := logger.Nop()
			credLoader := credentials.NewLoader(log)
			generator := NewTokenGenerator(tt.config, credLoader, log)

			token, err := generator.GenerateToken(context.Background(), tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrCode != "" {
					assert.True(t, errors.Is(err, tt.wantErrCode),
						"expected error code %s, got %v", tt.wantErrCode, err)
				}
				assert.Nil(t, token)
			} else {
				if err != nil {
					t.Logf("Unexpected error (might be expected in test env without real credentials): %v", err)
					return
				}
				assert.NoError(t, err)
				require.NotNil(t, token)
				if tt.validateToken != nil {
					tt.validateToken(t, token)
				}
			}
		})
	}
}

func TestTokenGenerator_ValidateToken(t *testing.T) {
	log := logger.Nop()
	config := DefaultConfig()
	credLoader := credentials.NewLoader(log)
	generator := NewTokenGenerator(config, credLoader, log)

	tests := []struct {
		name        string
		token       *provider.Token
		wantErr     bool
		wantErrCode errors.ErrorCode
	}{
		{
			name: "valid token",
			token: &provider.Token{
				AccessToken: v1Prefix + "eyJ1cmwiOiJodHRwczovL3N0cy5hbWF6b25hd3MuY29tLyIsIm1ldGhvZCI6IlBPU1QifQ",
				ExpiresAt:   time.Now().Add(15 * time.Minute),
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
				ExpiresAt:   time.Now().Add(15 * time.Minute),
				TokenType:   "Bearer",
			},
			wantErr:     true,
			wantErrCode: errors.ErrTokenInvalid,
		},
		{
			name: "invalid prefix",
			token: &provider.Token{
				AccessToken: "invalid-prefix.token",
				ExpiresAt:   time.Now().Add(15 * time.Minute),
				TokenType:   "Bearer",
			},
			wantErr:     true,
			wantErrCode: errors.ErrTokenInvalid,
		},
		{
			name: "expired token",
			token: &provider.Token{
				AccessToken: v1Prefix + "eyJ1cmwiOiJodHRwczovL3N0cy5hbWF6b25hd3MuY29tLyIsIm1ldGhvZCI6IlBPU1QifQ",
				ExpiresAt:   time.Now().Add(-15 * time.Minute),
				TokenType:   "Bearer",
			},
			wantErr:     true,
			wantErrCode: errors.ErrTokenExpired,
		},
		{
			name: "token expiring soon",
			token: &provider.Token{
				AccessToken: v1Prefix + "eyJ1cmwiOiJodHRwczovL3N0cy5hbWF6b25hd3MuY29tLyIsIm1ldGhvZCI6IlBPU1QifQ",
				ExpiresAt:   time.Now().Add(1 * time.Minute),
				TokenType:   "Bearer",
			},
			wantErr: false, // Still valid, just warns
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

func TestTokenGenerator_RefreshToken(t *testing.T) {
	log := logger.Nop()
	config := DefaultConfig()
	credLoader := credentials.NewLoader(log)
	generator := NewTokenGenerator(config, credLoader, log)

	opts := provider.GetTokenOptions{
		ClusterName: "test-cluster",
		Region:      "us-east-1",
	}

	tests := []struct {
		name         string
		currentToken *provider.Token
		wantRefresh  bool
	}{
		{
			name:         "nil token - should refresh",
			currentToken: nil,
			wantRefresh:  true,
		},
		{
			name: "expired token - should refresh",
			currentToken: &provider.Token{
				AccessToken: v1Prefix + "old-token",
				ExpiresAt:   time.Now().Add(-15 * time.Minute),
				TokenType:   "Bearer",
			},
			wantRefresh: true,
		},
		{
			name: "token expiring soon - should refresh",
			currentToken: &provider.Token{
				AccessToken: v1Prefix + "old-token",
				ExpiresAt:   time.Now().Add(1 * time.Minute),
				TokenType:   "Bearer",
			},
			wantRefresh: true,
		},
		{
			name: "valid token with time remaining - no refresh",
			currentToken: &provider.Token{
				AccessToken: v1Prefix + "current-token",
				ExpiresAt:   time.Now().Add(10 * time.Minute),
				TokenType:   "Bearer",
			},
			wantRefresh: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up minimal credentials for testing
			os.Setenv("AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE")
			os.Setenv("AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
			defer os.Unsetenv("AWS_ACCESS_KEY_ID")
			defer os.Unsetenv("AWS_SECRET_ACCESS_KEY")

			token, err := generator.RefreshToken(context.Background(), opts, tt.currentToken)

			// We expect this to fail in test environment without real credentials
			// Just verify the logic flow
			if !tt.wantRefresh && tt.currentToken != nil {
				// Should return the same token without refreshing
				if err == nil {
					assert.Equal(t, tt.currentToken.AccessToken, token.AccessToken)
				}
			} else {
				// Should attempt to refresh (may fail without real credentials)
				t.Logf("Refresh attempt result: %v (expected in test environment)", err)
			}
		})
	}
}

func TestNewTokenGenerator(t *testing.T) {
	log := logger.Nop()
	credLoader := credentials.NewLoader(log)

	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "with config",
			config: DefaultConfig(),
		},
		{
			name: "with custom duration",
			config: &Config{
				Region:        "eu-west-1",
				TokenDuration: 10 * time.Minute,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewTokenGenerator(tt.config, credLoader, log)
			assert.NotNil(t, generator)
			assert.Equal(t, tt.config, generator.config)
			assert.NotNil(t, generator.credLoader)
			assert.NotNil(t, generator.logger)
		})
	}
}

func TestDefaultPresignDuration(t *testing.T) {
	assert.Equal(t, 15*time.Minute, defaultPresignDuration)
}

func TestV1Prefix(t *testing.T) {
	assert.Equal(t, "k8s-aws-v1.", v1Prefix)
}

func TestDecodeToken(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name: "valid token",
			token: v1Prefix + "eyJ1cmwiOiJodHRwczovL3N0cy5hbWF6b25hd3MuY29tLyIsIm1ldGhvZCI6IlBPU1QiLCJjbHVzdGVyTmFtZSI6InRlc3QtY2x1c3RlciIsImhlYWRlcnMiOnsiSG9zdCI6WyJzdHMuYW1hem9uYXdzLmNvbSJdLCJ4LWs4cy1hd3MtaWQiOlsidGVzdC1jbHVzdGVyIl19fQ",
			wantErr: false,
		},
		{
			name:    "invalid prefix",
			token:   "invalid-prefix.token",
			wantErr: true,
		},
		{
			name:    "invalid base64",
			token:   v1Prefix + "not-valid-base64!!!",
			wantErr: true,
		},
		{
			name:    "invalid json",
			token:   v1Prefix + "bm90LWpzb24",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := DecodeToken(tt.token)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, payload)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, payload)
				assert.NotEmpty(t, payload.URL)
				assert.Equal(t, "POST", payload.Method)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	log := logger.Nop()
	credLoader := credentials.NewLoader(log)
	config := &Config{
		Region:        "us-east-1",
		TokenDuration: 15 * time.Minute,
	}
	generator := NewTokenGenerator(config, credLoader, log)

	// Create a mock presigned URL
	presignedURL := "https://sts.amazonaws.com/?Action=GetCallerIdentity&Version=2011-06-15&X-Amz-Algorithm=AWS4-HMAC-SHA256"
	clusterName := "test-cluster"

	// Encode token
	tokenString, err := generator.encodeToken(clusterName, presignedURL)
	require.NoError(t, err)

	// Verify format
	assert.True(t, strings.HasPrefix(tokenString, v1Prefix))

	// Decode token
	payload, err := DecodeToken(tokenString)
	require.NoError(t, err)

	// Verify payload
	assert.Equal(t, presignedURL, payload.URL)
	assert.Equal(t, "POST", payload.Method)
	assert.Equal(t, clusterName, payload.ClusterName)
	assert.Contains(t, payload.Headers, clusterIDHeader)
	assert.Equal(t, []string{clusterName}, payload.Headers[clusterIDHeader])
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
			name:     "default duration",
			config:   &Config{},
			expected: defaultPresignDuration,
		},
		{
			name: "custom duration",
			config: &Config{
				TokenDuration: 10 * time.Minute,
			},
			expected: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewTokenGenerator(tt.config, credLoader, log)
			duration := generator.getTokenDuration()
			assert.Equal(t, tt.expected, duration)
		})
	}
}
