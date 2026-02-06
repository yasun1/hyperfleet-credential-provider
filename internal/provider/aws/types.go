package aws

import (
	"time"
)

// Config holds AWS provider configuration
type Config struct {
	Region          string
	AccountID       string
	RoleARN         string
	CredentialsFile string
	TokenDuration   time.Duration
}

// DefaultConfig returns default AWS configuration
func DefaultConfig() *Config {
	return &Config{
		TokenDuration: 15 * time.Minute,
	}
}
