package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/credentials"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
)

const (
	// v1Prefix is the version prefix for EKS bearer tokens
	v1Prefix = "k8s-aws-v1."

	// clusterIDHeader is the header name for the cluster identifier
	clusterIDHeader = "x-k8s-aws-id"

	// defaultPresignDuration is the default duration for presigned URLs
	defaultPresignDuration = 15 * time.Minute
)

// TokenGenerator handles AWS STS token generation for EKS clusters
type TokenGenerator struct {
	config     *Config
	credLoader credentials.Loader
	logger     logger.Logger
}

// NewTokenGenerator creates a new AWS token generator
func NewTokenGenerator(config *Config, credLoader credentials.Loader, logger logger.Logger) *TokenGenerator {
	return &TokenGenerator{
		config:     config,
		credLoader: credLoader,
		logger:     logger,
	}
}

// GenerateToken generates a presigned STS token for EKS authentication
func (g *TokenGenerator) GenerateToken(ctx context.Context, opts provider.GetTokenOptions) (*provider.Token, error) {
	startTime := time.Now()

	g.logger.Debug("Starting AWS token generation",
		logger.String("cluster", opts.ClusterName),
		logger.String("region", opts.Region),
		logger.String("account_id", opts.AccountID),
	)

	// Validate cluster name
	if opts.ClusterName == "" {
		return nil, errors.New(
			errors.ErrInvalidArgument,
			"cluster name is required",
		).WithField("provider", "aws")
	}

	// Step 1: Load AWS credentials and create config
	awsConfig, err := g.loadAWSConfig(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Step 2: Create STS presigner
	stsClient := sts.NewFromConfig(awsConfig)
	presignClient := sts.NewPresignClient(stsClient)

	// Step 3: Create presigned GetCallerIdentity request
	presignedURL, err := g.createPresignedURL(ctx, presignClient, opts)
	if err != nil {
		return nil, err
	}

	// Step 4: Encode presigned URL as EKS token
	tokenString, err := g.encodeToken(opts.ClusterName, presignedURL)
	if err != nil {
		return nil, err
	}

	// Step 5: Create provider token
	expiresAt := time.Now().Add(g.getTokenDuration())
	token := &provider.Token{
		AccessToken: tokenString,
		ExpiresAt:   expiresAt,
		TokenType:   "Bearer",
	}

	duration := time.Since(startTime)
	g.logger.Info("AWS token generated successfully",
		logger.String("cluster", opts.ClusterName),
		logger.String("region", opts.Region),
		logger.Duration("duration_ms", duration.Milliseconds()),
		logger.String("expires_at", token.ExpiresAt.Format(time.RFC3339)),
		logger.Duration("expires_in_seconds", int64(token.ExpiresIn().Seconds())),
	)

	return token, nil
}

// loadAWSConfig loads AWS configuration from credentials and environment
func (g *TokenGenerator) loadAWSConfig(ctx context.Context, opts provider.GetTokenOptions) (aws.Config, error) {
	// Determine region
	region := opts.Region
	if region == "" && g.config.Region != "" {
		region = g.config.Region
	}

	// Load AWS credentials
	credOpts := credentials.AWSCredentialOptions{
		Region:         region,
		UseEnvironment: true,
	}

	creds, err := g.credLoader.LoadAWS(ctx, credOpts)
	if err != nil {
		return aws.Config{}, errors.Wrap(
			errors.ErrCredentialLoadFailed,
			err,
			"failed to load AWS credentials",
		).WithField("provider", "aws")
	}

	g.logger.Debug("AWS credentials loaded",
		logger.String("region", creds.Region),
		logger.Bool("has_session_token", creds.SessionToken != ""),
	)

	// Load AWS config with credentials
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     creds.AccessKeyID,
				SecretAccessKey: creds.SecretAccessKey,
				SessionToken:    creds.SessionToken,
			}, nil
		})),
	)
	if err != nil {
		return aws.Config{}, errors.Wrap(
			errors.ErrCredentialInvalid,
			err,
			"failed to create AWS config",
		).WithField("provider", "aws")
	}

	return cfg, nil
}

// createPresignedURL creates a presigned GetCallerIdentity URL for EKS authentication
func (g *TokenGenerator) createPresignedURL(ctx context.Context, presigner *sts.PresignClient, opts provider.GetTokenOptions) (string, error) {
	// Create GetCallerIdentity input
	input := &sts.GetCallerIdentityInput{}

	// Presign the request
	// Note: The cluster name will be encoded in the token payload, not as a header here
	presignResult, err := presigner.PresignGetCallerIdentity(ctx, input)
	if err != nil {
		return "", errors.Wrap(
			errors.ErrTokenGenerationFailed,
			err,
			"failed to presign GetCallerIdentity request",
		).WithFields(map[string]interface{}{
			"provider": "aws",
			"cluster":  opts.ClusterName,
			"region":   opts.Region,
		})
	}

	g.logger.Debug("Presigned URL created",
		logger.String("url_length", fmt.Sprintf("%d", len(presignResult.URL))),
		logger.String("method", presignResult.Method),
	)

	return presignResult.URL, nil
}

// encodeToken encodes the presigned URL and cluster name into an EKS bearer token
// Format: "k8s-aws-v1." + base64url(JSON payload)
func (g *TokenGenerator) encodeToken(clusterName string, presignedURL string) (string, error) {
	// Parse the presigned URL
	parsedURL, err := url.Parse(presignedURL)
	if err != nil {
		return "", errors.Wrap(
			errors.ErrTokenMalformed,
			err,
			"failed to parse presigned URL",
		).WithField("provider", "aws")
	}

	// Create the token payload
	// The payload contains the HTTP method, URL, headers, and body
	payload := &stsPresignedURLPayload{
		URL:        presignedURL,
		Method:     http.MethodPost,
		ClusterName: clusterName,
		Headers: map[string][]string{
			clusterIDHeader: {clusterName},
			"Host":         {parsedURL.Host},
		},
	}

	// Marshal to JSON
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", errors.Wrap(
			errors.ErrTokenMalformed,
			err,
			"failed to marshal token payload",
		).WithField("provider", "aws")
	}

	// Base64url encode (without padding)
	encoded := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Add version prefix
	token := v1Prefix + encoded

	g.logger.Debug("Token encoded",
		logger.String("token_length", fmt.Sprintf("%d", len(token))),
		logger.String("prefix", v1Prefix),
	)

	return token, nil
}

// stsPresignedURLPayload represents the payload for an EKS authentication token
type stsPresignedURLPayload struct {
	URL         string              `json:"url"`
	Method      string              `json:"method"`
	ClusterName string              `json:"clusterName"`
	Headers     map[string][]string `json:"headers"`
}

// getTokenDuration returns the configured token duration or default
func (g *TokenGenerator) getTokenDuration() time.Duration {
	if g.config.TokenDuration > 0 {
		return g.config.TokenDuration
	}
	return defaultPresignDuration
}

// ValidateToken validates that a token is valid and not expired
func (g *TokenGenerator) ValidateToken(token *provider.Token) error {
	if token == nil {
		return errors.New(
			errors.ErrTokenInvalid,
			"token is nil",
		).WithField("provider", "aws")
	}

	if token.AccessToken == "" {
		return errors.New(
			errors.ErrTokenInvalid,
			"access token is empty",
		).WithField("provider", "aws")
	}

	// Validate token format
	if !strings.HasPrefix(token.AccessToken, v1Prefix) {
		return errors.New(
			errors.ErrTokenInvalid,
			"token does not have expected prefix",
		).WithFields(map[string]interface{}{
			"provider":        "aws",
			"expected_prefix": v1Prefix,
		})
	}

	if token.IsExpired() {
		return errors.New(
			errors.ErrTokenExpired,
			"token has expired",
		).WithFields(map[string]interface{}{
			"provider":   "aws",
			"expires_at": token.ExpiresAt.Format(time.RFC3339),
		})
	}

	// Warn if token expires soon (less than 2 minutes for AWS's shorter duration)
	if token.ExpiresIn() < 2*time.Minute {
		g.logger.Warn("Token expires soon",
			logger.String("provider", "aws"),
			logger.Duration("expires_in_seconds", int64(token.ExpiresIn().Seconds())),
		)
	}

	return nil
}

// RefreshToken refreshes an expired or soon-to-expire token
func (g *TokenGenerator) RefreshToken(ctx context.Context, opts provider.GetTokenOptions, currentToken *provider.Token) (*provider.Token, error) {
	// Check if refresh is needed
	if currentToken != nil && !currentToken.IsExpired() {
		// For AWS, refresh if less than 2 minutes remaining (due to shorter 15min duration)
		if currentToken.ExpiresIn() > 2*time.Minute {
			g.logger.Debug("Token still valid, no refresh needed",
				logger.String("provider", "aws"),
				logger.Duration("expires_in_seconds", int64(currentToken.ExpiresIn().Seconds())),
			)
			return currentToken, nil
		}
	}

	g.logger.Info("Refreshing AWS token",
		logger.String("cluster", opts.ClusterName),
		logger.Bool("expired", currentToken == nil || currentToken.IsExpired()),
	)

	// Generate new token
	return g.GenerateToken(ctx, opts)
}

// DecodeToken decodes an EKS token to extract the payload (for debugging/validation)
func DecodeToken(token string) (*stsPresignedURLPayload, error) {
	// Remove prefix
	if !strings.HasPrefix(token, v1Prefix) {
		return nil, fmt.Errorf("invalid token prefix")
	}

	encoded := strings.TrimPrefix(token, v1Prefix)

	// Base64url decode
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	// Unmarshal JSON
	var payload stsPresignedURLPayload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token payload: %w", err)
	}

	return &payload, nil
}
