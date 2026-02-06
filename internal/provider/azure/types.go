package azure

import (
	"time"
)

// Config holds Azure provider configuration
type Config struct {
	SubscriptionID  string
	TenantID        string
	ResourceGroup   string
	CredentialsFile string
	TokenDuration   time.Duration
}

// DefaultConfig returns default Azure configuration
func DefaultConfig() *Config {
	return &Config{
		TokenDuration: 1 * time.Hour,
	}
}
