package config

import (
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/errors"
)

// LoadOption is a functional option for loading configuration
type LoadOption func(*loadOptions)

type loadOptions struct {
	configFile string
	fromEnv    bool
}

// WithConfigFile specifies the config file path
func WithConfigFile(path string) LoadOption {
	return func(o *loadOptions) {
		o.configFile = path
	}
}

// WithEnv enables environment variable overrides
func WithEnv() LoadOption {
	return func(o *loadOptions) {
		o.fromEnv = true
	}
}

// Load loads configuration with the given options
func Load(opts ...LoadOption) (*Config, error) {
	options := &loadOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Start with default config
	config := DefaultConfig()

	// Load from file if specified
	if options.configFile != "" {
		fileConfig, err := loadFromFile(options.configFile)
		if err != nil {
			return nil, err
		}
		config.Merge(fileConfig)
	}

	// Override with environment variables if enabled
	if options.fromEnv {
		envConfig := loadFromEnv()
		config.Merge(envConfig)
	}

	// Validate final configuration
	if err := Validate(config); err != nil {
		return nil, err
	}

	return config, nil
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(
			errors.ErrConfigLoadFailed,
			err,
			"failed to read config file",
		).WithField("path", path)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.Wrap(
			errors.ErrConfigInvalid,
			err,
			"failed to parse config file",
		).WithField("path", path)
	}

	return &config, nil
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv() *Config {
	config := &Config{
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", ""),
			Format: getEnv("LOG_FORMAT", ""),
		},
		Provider: ProviderConfig{
			Name:        getEnv("PROVIDER", ""),
			Region:      getEnv("PROVIDER_REGION", ""),
			ClusterName: getEnv("CLUSTER_NAME", ""),
			Timeout:     getDurationEnv("PROVIDER_TIMEOUT", 0),
		},
		Health: HealthConfig{
			Enabled:       getBoolEnv("HEALTH_ENABLED", true),
			Port:          getIntEnv("HEALTH_PORT", 0),
			ReadinessPath: getEnv("HEALTH_READINESS_PATH", ""),
			LivenessPath:  getEnv("HEALTH_LIVENESS_PATH", ""),
		},
		Metrics: MetricsConfig{
			Enabled: getBoolEnv("METRICS_ENABLED", true),
			Port:    getIntEnv("METRICS_PORT", 0),
			Path:    getEnv("METRICS_PATH", ""),
		},
	}

	// Load GCP config
	if gcpProjectID := getEnv("GCP_PROJECT_ID", ""); gcpProjectID != "" {
		config.Provider.GCP = &GCPConfig{
			ProjectID:       gcpProjectID,
			CredentialsFile: getEnv("GOOGLE_APPLICATION_CREDENTIALS", ""),
			TokenDuration:   getDurationEnv("GCP_TOKEN_DURATION", 0),
		}
	}

	// Load AWS config
	if getEnv("AWS_REGION", "") != "" || getEnv("AWS_ACCESS_KEY_ID", "") != "" {
		config.Provider.AWS = &AWSConfig{
			AccountID:     getEnv("AWS_ACCOUNT_ID", ""),
			RoleARN:       getEnv("AWS_ROLE_ARN", ""),
			TokenDuration: getDurationEnv("AWS_TOKEN_DURATION", 0),
		}
	}

	// Load Azure config
	if azureSubscriptionID := getEnv("AZURE_SUBSCRIPTION_ID", ""); azureSubscriptionID != "" {
		config.Provider.Azure = &AzureConfig{
			SubscriptionID: azureSubscriptionID,
			TenantID:       getEnv("AZURE_TENANT_ID", ""),
			ResourceGroup:  getEnv("AZURE_RESOURCE_GROUP", ""),
			TokenDuration:  getDurationEnv("AZURE_TOKEN_DURATION", 0),
		}
	}

	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv gets an integer environment variable with a default value
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getBoolEnv gets a boolean environment variable with a default value
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// getDurationEnv gets a duration environment variable with a default value
// Expects value in seconds
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}

// FromFlags creates a config from command-line flags (used by CLI)
func FromFlags(
	provider string,
	clusterName string,
	region string,
	logLevel string,
	logFormat string,
) *Config {
	config := DefaultConfig()

	config.Provider.Name = provider
	config.Provider.ClusterName = clusterName
	config.Provider.Region = region

	if logLevel != "" {
		config.Log.Level = logLevel
	}
	if logFormat != "" {
		config.Log.Format = logFormat
	}

	return config
}
