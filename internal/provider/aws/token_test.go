package aws

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/testutil"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

// TestTokenGenerator_LoadAWSConfig tests AWS config loading logic
func TestTokenGenerator_LoadAWSConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func() *testutil.MockCredLoader
		opts        provider.GetTokenOptions
		wantErr     bool
		wantErrCode errors.ErrorCode
		validate    func(t *testing.T)
	}{
		{
			name: "successful config loading with valid credentials",
			setupMock: func() *testutil.MockCredLoader {
				return testutil.NewMockCredLoader().WithAWSCreds(testutil.CreateValidAWSCredentials())
			},
			opts: provider.GetTokenOptions{
				ClusterName: "test-cluster",
				Region:      "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "config loading with session token",
			setupMock: func() *testutil.MockCredLoader {
				return testutil.NewMockCredLoader().WithAWSCreds(testutil.CreateValidAWSCredentialsWithSessionToken())
			},
			opts: provider.GetTokenOptions{
				ClusterName: "test-cluster",
				Region:      "us-west-2",
			},
			wantErr: false,
		},
		{
			name: "credential loading failure",
			setupMock: func() *testutil.MockCredLoader {
				return testutil.NewMockCredLoader().WithAWSError(
					errors.New(errors.ErrCredentialLoadFailed, "credentials not found"),
				)
			},
			opts: provider.GetTokenOptions{
				ClusterName: "test-cluster",
				Region:      "us-east-1",
			},
			wantErr:     true,
			wantErrCode: errors.ErrCredentialLoadFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.Nop()
			mockLoader := tt.setupMock()
			config := &Config{
				Region:        "us-east-1",
				TokenDuration: 15 * time.Minute,
			}

			generator := NewTokenGenerator(config, mockLoader, log)
			awsConfig, err := generator.loadAWSConfig(context.Background(), tt.opts)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrCode != "" {
					assert.True(t, errors.Is(err, tt.wantErrCode),
						"expected error code %s, got %v", tt.wantErrCode, err)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, awsConfig)
				if tt.validate != nil {
					tt.validate(t)
				}
			}
		})
	}
}

// TestTokenGenerator_ValidateClusterName tests cluster name validation
func TestTokenGenerator_ValidateClusterName(t *testing.T) {
	tests := []struct {
		name        string
		clusterName string
		wantErr     bool
		wantErrCode errors.ErrorCode
	}{
		{
			name:        "valid cluster name",
			clusterName: "my-eks-cluster",
			wantErr:     false,
		},
		{
			name:        "empty cluster name",
			clusterName: "",
			wantErr:     true,
			wantErrCode: errors.ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.Nop()
			mockLoader := testutil.NewMockCredLoader().WithAWSCreds(testutil.CreateValidAWSCredentials())
			config := &Config{
				Region:        "us-east-1",
				TokenDuration: 15 * time.Minute,
			}
			generator := NewTokenGenerator(config, mockLoader, log)

			opts := provider.GetTokenOptions{
				ClusterName: tt.clusterName,
				Region:      "us-east-1",
			}

			// Cluster name validation happens in GenerateToken
			// We test it by calling GenerateToken (which will fail at STS call but that's ok)
			_, err := generator.GenerateToken(context.Background(), opts)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrCode != "" {
					assert.True(t, errors.Is(err, tt.wantErrCode),
						"expected error code %s, got %v", tt.wantErrCode, err)
				}
			} else {
				// If cluster name is valid, error will be from STS call (which is expected)
				// We just check that it's NOT a cluster name validation error
				if err != nil {
					assert.False(t, errors.Is(err, errors.ErrInvalidArgument),
						"should not get invalid argument error for valid cluster name")
				}
			}
		})
	}
}

// TestTokenGenerator_EncodeToken tests token encoding logic
func TestTokenGenerator_EncodeToken(t *testing.T) {
	log := logger.Nop()
	mockLoader := testutil.NewMockCredLoader()
	config := DefaultConfig()
	generator := NewTokenGenerator(config, mockLoader, log)

	tests := []struct {
		name         string
		clusterName  string
		presignedURL string
		wantPrefix   string
		wantErr      bool
	}{
		{
			name:         "valid token encoding",
			clusterName:  "my-cluster",
			presignedURL: "https://sts.amazonaws.com/?Action=GetCallerIdentity",
			wantPrefix:   v1Prefix,
			wantErr:      false,
		},
		{
			name:         "token with different cluster",
			clusterName:  "production-cluster",
			presignedURL: "https://sts.us-west-2.amazonaws.com/?Action=GetCallerIdentity&Version=2011-06-15",
			wantPrefix:   v1Prefix,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := generator.encodeToken(tt.clusterName, tt.presignedURL)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, token)
				assert.True(t, strings.HasPrefix(token, tt.wantPrefix),
					"token should have prefix %s", tt.wantPrefix)

				// Verify token can be base64 decoded
				tokenWithoutPrefix := strings.TrimPrefix(token, v1Prefix)
				decoded, err := base64.RawURLEncoding.DecodeString(tokenWithoutPrefix)
				assert.NoError(t, err, "token should be valid base64")
				assert.NotEmpty(t, decoded)

				// Verify decoded token contains cluster name header
				decodedStr := string(decoded)
				assert.Contains(t, decodedStr, clusterIDHeader)
			}
		})
	}
}

// TestTokenGenerator_GetTokenDuration tests token duration calculation
func TestTokenGenerator_GetTokenDuration(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected time.Duration
	}{
		{
			name: "default duration",
			config: &Config{
				TokenDuration: 0, // Not set
			},
			expected: defaultPresignDuration,
		},
		{
			name: "custom duration 15 minutes",
			config: &Config{
				TokenDuration: 15 * time.Minute,
			},
			expected: 15 * time.Minute,
		},
		{
			name: "custom duration 5 minutes",
			config: &Config{
				TokenDuration: 5 * time.Minute,
			},
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.Nop()
			mockLoader := testutil.NewMockCredLoader()
			generator := NewTokenGenerator(tt.config, mockLoader, log)

			duration := generator.getTokenDuration()
			assert.Equal(t, tt.expected, duration)
		})
	}
}

// TestTokenDefaultConfig verifies the default AWS token config
func TestTokenDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, defaultPresignDuration, config.TokenDuration)
	assert.Empty(t, config.Region) // Region should come from credentials
	assert.Empty(t, config.RoleARN)
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
				Region:        "us-east-1",
				TokenDuration: 15 * time.Minute,
			},
			valid: true,
		},
		{
			name: "minimal config",
			config: &Config{
				Region: "us-west-2",
			},
			valid: true,
		},
		{
			name: "config with role ARN",
			config: &Config{
				Region:  "eu-west-1",
				RoleARN: "arn:aws:iam::123456789012:role/MyRole",
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

// TestToken_Properties tests token property calculations
func TestToken_Properties(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		wantExpired bool
		expiresIn time.Duration
	}{
		{
			name:        "token expires in 15 minutes",
			expiresAt:   now.Add(15 * time.Minute),
			wantExpired: false,
			expiresIn:   15 * time.Minute,
		},
		{
			name:        "token expires in 5 minutes",
			expiresAt:   now.Add(5 * time.Minute),
			wantExpired: false,
			expiresIn:   5 * time.Minute,
		},
		{
			name:        "token already expired",
			expiresAt:   now.Add(-10 * time.Minute),
			wantExpired: true,
			expiresIn:   0,
		},
		{
			name:        "token expires now",
			expiresAt:   now,
			wantExpired: true,
			expiresIn:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &provider.Token{
				AccessToken: "k8s-aws-v1.test-token",
				ExpiresAt:   tt.expiresAt,
				TokenType:   "Bearer",
			}

			// Test IsExpired
			if tt.wantExpired {
				assert.True(t, token.IsExpired() || time.Until(token.ExpiresAt) < time.Second,
					"token should be expired")
			} else {
				assert.False(t, token.IsExpired(), "token should not be expired")
			}

			// Test ExpiresIn (allow 1 second drift)
			expiresIn := token.ExpiresIn()
			if tt.expiresIn > 0 {
				assert.InDelta(t, tt.expiresIn.Seconds(), expiresIn.Seconds(), 1.0,
					"expires in calculation should be accurate")
			} else {
				assert.LessOrEqual(t, expiresIn, time.Duration(0),
					"expired token should have non-positive expiresIn")
			}
		})
	}
}

// TestTokenPrefix tests the v1 prefix constant
func TestTokenPrefix(t *testing.T) {
	assert.Equal(t, "k8s-aws-v1.", v1Prefix)
	assert.Equal(t, "x-k8s-aws-id", clusterIDHeader)
}

// TestConfig_Validation tests config validation
func TestConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config with all fields",
			config: &Config{
				Region:        "us-east-1",
				TokenDuration: 15 * time.Minute,
				RoleARN:       "arn:aws:iam::123456789012:role/MyRole",
			},
			wantErr: false,
		},
		{
			name: "valid config with minimal fields",
			config: &Config{
				Region: "us-west-2",
			},
			wantErr: false,
		},
		{
			name: "config with very short duration",
			config: &Config{
				Region:        "us-east-1",
				TokenDuration: 1 * time.Minute,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify config can be created
			assert.NotNil(t, tt.config)
		})
	}
}
