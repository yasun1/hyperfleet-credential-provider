package gcp

import (
	"time"
)

// Config holds GCP provider configuration
type Config struct {
	ProjectID         string
	CredentialsFile   string
	TokenDuration     time.Duration
	Scopes            []string
}

// DefaultScopes returns the default OAuth scopes for GKE access
func DefaultScopes() []string {
	return []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/userinfo.email",
	}
}

// DefaultConfig returns default GCP configuration
func DefaultConfig() *Config {
	return &Config{
		TokenDuration: 1 * time.Hour,
		Scopes:        DefaultScopes(),
	}
}
