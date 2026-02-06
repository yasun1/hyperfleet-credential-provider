package config

import (
	"time"
)

// Config represents the complete provider configuration
type Config struct {
	// Log configuration
	Log LogConfig `yaml:"log" validate:"required"`

	// Provider configuration
	Provider ProviderConfig `yaml:"provider" validate:"required"`

	// Health server configuration
	Health HealthConfig `yaml:"health"`

	// Metrics configuration
	Metrics MetricsConfig `yaml:"metrics"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	// Level is the log level (debug, info, warn, error)
	Level string `yaml:"level" validate:"required,oneof=debug info warn error"`

	// Format is the log format (json, console)
	Format string `yaml:"format" validate:"required,oneof=json console"`
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	// Name is the cloud provider name (gcp, aws, azure)
	Name string `yaml:"name" validate:"required,oneof=gcp aws azure"`

	// Region is the cloud region
	Region string `yaml:"region"`

	// ClusterName is the Kubernetes cluster name
	ClusterName string `yaml:"cluster_name" validate:"required"`

	// GCP-specific configuration
	GCP *GCPConfig `yaml:"gcp,omitempty"`

	// AWS-specific configuration
	AWS *AWSConfig `yaml:"aws,omitempty"`

	// Azure-specific configuration
	Azure *AzureConfig `yaml:"azure,omitempty"`

	// Timeout for provider operations
	Timeout time.Duration `yaml:"timeout" validate:"min=0"`
}

// GCPConfig holds GCP-specific configuration
type GCPConfig struct {
	// ProjectID is the GCP project ID
	ProjectID string `yaml:"project_id" validate:"required"`

	// CredentialsFile is the path to the service account JSON file
	CredentialsFile string `yaml:"credentials_file"`

	// TokenDuration is the token expiration duration
	TokenDuration time.Duration `yaml:"token_duration"`
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	// AccountID is the AWS account ID (optional)
	AccountID string `yaml:"account_id"`

	// RoleARN is the IAM role ARN to assume (optional)
	RoleARN string `yaml:"role_arn"`

	// TokenDuration is the token expiration duration
	TokenDuration time.Duration `yaml:"token_duration"`
}

// AzureConfig holds Azure-specific configuration
type AzureConfig struct {
	// SubscriptionID is the Azure subscription ID
	SubscriptionID string `yaml:"subscription_id" validate:"required"`

	// TenantID is the Azure tenant ID
	TenantID string `yaml:"tenant_id" validate:"required"`

	// ResourceGroup is the cluster resource group (optional)
	ResourceGroup string `yaml:"resource_group"`

	// TokenDuration is the token expiration duration
	TokenDuration time.Duration `yaml:"token_duration"`
}

// HealthConfig holds health server configuration
type HealthConfig struct {
	// Enabled determines if health server is enabled
	Enabled bool `yaml:"enabled"`

	// Port is the health server port
	Port int `yaml:"port" validate:"min=0,max=65535"`

	// ReadinessProbe path
	ReadinessPath string `yaml:"readiness_path"`

	// LivenessProbe path
	LivenessPath string `yaml:"liveness_path"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	// Enabled determines if metrics are enabled
	Enabled bool `yaml:"enabled"`

	// Port is the metrics server port
	Port int `yaml:"port" validate:"min=0,max=65535"`

	// Path is the metrics endpoint path
	Path string `yaml:"path"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
		Provider: ProviderConfig{
			Timeout: 30 * time.Second,
		},
		Health: HealthConfig{
			Enabled:       true,
			Port:          8080,
			ReadinessPath: "/readyz",
			LivenessPath:  "/healthz",
		},
		Metrics: MetricsConfig{
			Enabled: true,
			Port:    8080,
			Path:    "/metrics",
		},
	}
}

// Merge merges the given config into this config
// Non-zero values from other take precedence
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}

	// Merge log config
	if other.Log.Level != "" {
		c.Log.Level = other.Log.Level
	}
	if other.Log.Format != "" {
		c.Log.Format = other.Log.Format
	}

	// Merge provider config
	if other.Provider.Name != "" {
		c.Provider.Name = other.Provider.Name
	}
	if other.Provider.Region != "" {
		c.Provider.Region = other.Provider.Region
	}
	if other.Provider.ClusterName != "" {
		c.Provider.ClusterName = other.Provider.ClusterName
	}
	if other.Provider.Timeout > 0 {
		c.Provider.Timeout = other.Provider.Timeout
	}

	// Merge provider-specific configs
	if other.Provider.GCP != nil {
		if c.Provider.GCP == nil {
			c.Provider.GCP = &GCPConfig{}
		}
		if other.Provider.GCP.ProjectID != "" {
			c.Provider.GCP.ProjectID = other.Provider.GCP.ProjectID
		}
		if other.Provider.GCP.CredentialsFile != "" {
			c.Provider.GCP.CredentialsFile = other.Provider.GCP.CredentialsFile
		}
		if other.Provider.GCP.TokenDuration > 0 {
			c.Provider.GCP.TokenDuration = other.Provider.GCP.TokenDuration
		}
	}

	if other.Provider.AWS != nil {
		if c.Provider.AWS == nil {
			c.Provider.AWS = &AWSConfig{}
		}
		if other.Provider.AWS.AccountID != "" {
			c.Provider.AWS.AccountID = other.Provider.AWS.AccountID
		}
		if other.Provider.AWS.RoleARN != "" {
			c.Provider.AWS.RoleARN = other.Provider.AWS.RoleARN
		}
		if other.Provider.AWS.TokenDuration > 0 {
			c.Provider.AWS.TokenDuration = other.Provider.AWS.TokenDuration
		}
	}

	if other.Provider.Azure != nil {
		if c.Provider.Azure == nil {
			c.Provider.Azure = &AzureConfig{}
		}
		if other.Provider.Azure.SubscriptionID != "" {
			c.Provider.Azure.SubscriptionID = other.Provider.Azure.SubscriptionID
		}
		if other.Provider.Azure.TenantID != "" {
			c.Provider.Azure.TenantID = other.Provider.Azure.TenantID
		}
		if other.Provider.Azure.ResourceGroup != "" {
			c.Provider.Azure.ResourceGroup = other.Provider.Azure.ResourceGroup
		}
		if other.Provider.Azure.TokenDuration > 0 {
			c.Provider.Azure.TokenDuration = other.Provider.Azure.TokenDuration
		}
	}

	// Merge health config
	c.Health.Enabled = other.Health.Enabled
	if other.Health.Port > 0 {
		c.Health.Port = other.Health.Port
	}
	if other.Health.ReadinessPath != "" {
		c.Health.ReadinessPath = other.Health.ReadinessPath
	}
	if other.Health.LivenessPath != "" {
		c.Health.LivenessPath = other.Health.LivenessPath
	}

	// Merge metrics config
	c.Metrics.Enabled = other.Metrics.Enabled
	if other.Metrics.Port > 0 {
		c.Metrics.Port = other.Metrics.Port
	}
	if other.Metrics.Path != "" {
		c.Metrics.Path = other.Metrics.Path
	}
}
