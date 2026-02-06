package gcp

import (
	"context"
	"encoding/json"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/credentials"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/errors"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

// TokenGenerator handles GCP OAuth2 token generation for GKE clusters
type TokenGenerator struct {
	config     *Config
	credLoader credentials.Loader
	logger     logger.Logger
}

// NewTokenGenerator creates a new GCP token generator
func NewTokenGenerator(config *Config, credLoader credentials.Loader, logger logger.Logger) *TokenGenerator {
	return &TokenGenerator{
		config:     config,
		credLoader: credLoader,
		logger:     logger,
	}
}

// GenerateToken generates an OAuth2 access token for GKE authentication
func (g *TokenGenerator) GenerateToken(ctx context.Context, opts provider.GetTokenOptions) (*provider.Token, error) {
	startTime := time.Now()

	g.logger.Debug("Starting GCP token generation",
		logger.String("cluster", opts.ClusterName),
		logger.String("project", opts.ProjectID),
		logger.String("region", opts.Region),
	)

	// Step 1: Load GCP credentials
	creds, err := g.loadCredentials(ctx)
	if err != nil {
		return nil, err
	}

	g.logger.Debug("Credentials loaded",
		logger.String("client_email", creds.ClientEmail),
		logger.String("project_id", creds.ProjectID),
	)

	// Step 2: Create OAuth2 token source
	tokenSource, err := g.createTokenSource(ctx, creds)
	if err != nil {
		return nil, err
	}

	// Step 3: Get OAuth2 token
	oauth2Token, err := tokenSource.Token()
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrTokenGenerationFailed,
			err,
			"failed to get OAuth2 token from token source",
		).WithFields(map[string]interface{}{
			"provider": "gcp",
			"cluster":  opts.ClusterName,
			"project":  opts.ProjectID,
		})
	}

	// Validate token
	if oauth2Token.AccessToken == "" {
		return nil, errors.New(
			errors.ErrTokenInvalid,
			"OAuth2 token is empty",
		).WithField("provider", "gcp")
	}

	// Step 4: Create provider token
	token := &provider.Token{
		AccessToken: oauth2Token.AccessToken,
		ExpiresAt:   oauth2Token.Expiry,
		TokenType:   oauth2Token.TokenType,
	}

	// Default to Bearer if not specified
	if token.TokenType == "" {
		token.TokenType = "Bearer"
	}

	duration := time.Since(startTime)
	g.logger.Info("GCP token generated successfully",
		logger.String("cluster", opts.ClusterName),
		logger.String("project", opts.ProjectID),
		logger.Duration("duration_ms", duration.Milliseconds()),
		logger.String("expires_at", token.ExpiresAt.Format(time.RFC3339)),
		logger.Duration("expires_in_seconds", int64(token.ExpiresIn().Seconds())),
	)

	return token, nil
}

// loadCredentials loads GCP service account credentials
func (g *TokenGenerator) loadCredentials(ctx context.Context) (*credentials.GCPCredentials, error) {
	creds, err := g.credLoader.LoadGCP(ctx, g.config.CredentialsFile)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialLoadFailed,
			err,
			"failed to load GCP credentials",
		).WithField("provider", "gcp")
	}

	// Validate project ID matches if specified in config
	if g.config.ProjectID != "" && creds.ProjectID != g.config.ProjectID {
		g.logger.Warn("Project ID mismatch between config and credentials",
			logger.String("config_project", g.config.ProjectID),
			logger.String("creds_project", creds.ProjectID),
		)
	}

	return creds, nil
}

// createTokenSource creates an OAuth2 token source from GCP credentials
func (g *TokenGenerator) createTokenSource(ctx context.Context, creds *credentials.GCPCredentials) (oauth2.TokenSource, error) {
	// Convert credentials struct to JSON bytes
	credsJSON, err := json.Marshal(creds)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialMalformed,
			err,
			"failed to marshal GCP credentials to JSON",
		).WithField("provider", "gcp")
	}

	// Create credentials from JSON with appropriate scopes
	googleCreds, err := google.CredentialsFromJSON(ctx, credsJSON, g.config.Scopes...)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrCredentialInvalid,
			err,
			"failed to create Google credentials from JSON",
		).WithFields(map[string]interface{}{
			"provider": "gcp",
			"scopes":   g.config.Scopes,
		})
	}

	g.logger.Debug("Token source created",
		logger.String("project_id", googleCreds.ProjectID),
		logger.Int("num_scopes", len(g.config.Scopes)),
	)

	return googleCreds.TokenSource, nil
}

// ValidateToken validates that a token is valid and not expired
func (g *TokenGenerator) ValidateToken(token *provider.Token) error {
	if token == nil {
		return errors.New(
			errors.ErrTokenInvalid,
			"token is nil",
		).WithField("provider", "gcp")
	}

	if token.AccessToken == "" {
		return errors.New(
			errors.ErrTokenInvalid,
			"access token is empty",
		).WithField("provider", "gcp")
	}

	if token.IsExpired() {
		return errors.New(
			errors.ErrTokenExpired,
			"token has expired",
		).WithFields(map[string]interface{}{
			"provider":   "gcp",
			"expires_at": token.ExpiresAt.Format(time.RFC3339),
		})
	}

	// Warn if token expires soon (less than 5 minutes)
	if token.ExpiresIn() < 5*time.Minute {
		g.logger.Warn("Token expires soon",
			logger.String("provider", "gcp"),
			logger.Duration("expires_in_seconds", int64(token.ExpiresIn().Seconds())),
		)
	}

	return nil
}

// RefreshToken refreshes an expired or soon-to-expire token
func (g *TokenGenerator) RefreshToken(ctx context.Context, opts provider.GetTokenOptions, currentToken *provider.Token) (*provider.Token, error) {
	// Check if refresh is needed
	if currentToken != nil && !currentToken.IsExpired() {
		// Token is still valid, check if it's close to expiring
		if currentToken.ExpiresIn() > 5*time.Minute {
			g.logger.Debug("Token still valid, no refresh needed",
				logger.String("provider", "gcp"),
				logger.Duration("expires_in_seconds", int64(currentToken.ExpiresIn().Seconds())),
			)
			return currentToken, nil
		}
	}

	g.logger.Info("Refreshing GCP token",
		logger.String("cluster", opts.ClusterName),
		logger.Bool("expired", currentToken == nil || currentToken.IsExpired()),
	)

	// Generate new token
	return g.GenerateToken(ctx, opts)
}
