package common

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/aws"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/azure"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/gcp"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

type Flags struct {
	LogLevel        string
	LogFormat       string
	CredentialsFile string

	ProviderName   string
	ClusterName    string
	Region         string
	ProjectID      string
	AccountID      string
	SubscriptionID string
	TenantID       string
	ResourceGroup  string
	TokenDuration  string
}

// InitViper initializes Viper for environment variable support
func InitViper() {
	viper.SetEnvPrefix("HFCP")

	// Replace hyphens with underscores in environment variables
	// e.g., --credentials-file -> HFCP_CREDENTIALS_FILE
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Automatically bind environment variables
	viper.AutomaticEnv()
}

// BindPersistentFlags binds persistent flags from root command to Viper
func BindPersistentFlags(cmd *cobra.Command) {
	viper.BindPFlags(cmd.PersistentFlags())
}

// BindCommandFlags binds command-specific flags to Viper
func BindCommandFlags(cmd *cobra.Command) error {
	// Bind local flags (specific to this command)
	return viper.BindPFlags(cmd.Flags())
}

// BindFlagsToViper binds command flags to Viper values
// This ensures environment variables are read if flags are not provided
func BindFlagsToViper(flags *Flags) {
	// Global flags - read from viper if not explicitly set via command line
	if !isFlagSetExplicitly("log-level") {
		flags.LogLevel = viper.GetString("log-level")
	}
	if !isFlagSetExplicitly("log-format") {
		flags.LogFormat = viper.GetString("log-format")
	}
	if !isFlagSetExplicitly("credentials-file") {
		flags.CredentialsFile = viper.GetString("credentials-file")
	}

	// Provider flags
	if !isFlagSetExplicitly("provider") {
		flags.ProviderName = viper.GetString("provider")
	}
	if !isFlagSetExplicitly("cluster-name") {
		flags.ClusterName = viper.GetString("cluster-name")
	}
	if !isFlagSetExplicitly("region") {
		flags.Region = viper.GetString("region")
	}
	if !isFlagSetExplicitly("project-id") {
		flags.ProjectID = viper.GetString("project-id")
	}
	if !isFlagSetExplicitly("account-id") {
		flags.AccountID = viper.GetString("account-id")
	}
	if !isFlagSetExplicitly("subscription-id") {
		flags.SubscriptionID = viper.GetString("subscription-id")
	}
	if !isFlagSetExplicitly("tenant-id") {
		flags.TenantID = viper.GetString("tenant-id")
	}
	if !isFlagSetExplicitly("resource-group") {
		flags.ResourceGroup = viper.GetString("resource-group")
	}
	if !isFlagSetExplicitly("token-duration") {
		flags.TokenDuration = viper.GetString("token-duration")
	}
}

// isFlagSetExplicitly checks if a flag was set explicitly on the command line
// If viper.IsSet returns true but the value equals the default, it was from env/config
func isFlagSetExplicitly(flagName string) bool {
	// This is a simplification - in practice we'd check the cobra command's flags
	// For now, we always prefer viper values (env vars take precedence)
	return false
}

func CreateLogger(flags *Flags) (logger.Logger, error) {
	var level logger.Level
	switch flags.LogLevel {
	case "debug":
		level = logger.DebugLevel
	case "info":
		level = logger.InfoLevel
	case "warn":
		level = logger.WarnLevel
	case "error":
		level = logger.ErrorLevel
	default:
		level = logger.InfoLevel
	}

	var format logger.Format
	switch flags.LogFormat {
	case "json":
		format = logger.JSONFormat
	case "console":
		format = logger.ConsoleFormat
	default:
		format = logger.JSONFormat
	}

	return logger.New(logger.Config{
		Level:  level,
		Format: format,
		Output: os.Stderr,
	})
}

func CreateProvider(flags *Flags, log logger.Logger) (provider.Provider, error) {
	switch flags.ProviderName {
	case "gcp":
		config := &gcp.Config{
			ProjectID:       flags.ProjectID,
			CredentialsFile: flags.CredentialsFile,
			TokenDuration:   1 * time.Hour,
			Scopes:          gcp.DefaultScopes(),
		}
		return gcp.NewProvider(config, log)

	case "aws":
		config := &aws.Config{
			Region:          flags.Region,
			CredentialsFile: flags.CredentialsFile,
			TokenDuration:   15 * time.Minute,
		}
		return aws.NewProvider(config, log)

	case "azure":
		config := &azure.Config{
			TenantID:        flags.TenantID,
			SubscriptionID:  flags.SubscriptionID,
			CredentialsFile: flags.CredentialsFile,
			TokenDuration:   1 * time.Hour,
		}
		return azure.NewProvider(config, log)

	default:
		return nil, fmt.Errorf("unsupported provider: %s (must be one of: gcp, aws, azure)", flags.ProviderName)
	}
}

func SetupSignalHandler() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	return ctx, cancel
}

func GetCredentialsPath(flags *Flags) string {
	if flags.CredentialsFile != "" {
		return flags.CredentialsFile
	}

	switch flags.ProviderName {
	case "gcp":
		return "/vault/secrets/gcp-sa.json"
	case "aws":
		return "/vault/secrets/aws-credentials"
	case "azure":
		return "/vault/secrets/azure-credentials.json"
	default:
		return "/vault/secrets/credentials"
	}
}

func ParseTokenDuration(flags *Flags) (time.Duration, error) {
	if flags.TokenDuration != "" {
		duration, err := time.ParseDuration(flags.TokenDuration)
		if err != nil {
			return 0, fmt.Errorf("invalid token duration format: %w (examples: 1h, 30m, 900s)", err)
		}
		if duration <= 0 {
			return 0, fmt.Errorf("token duration must be positive")
		}
		return duration, nil
	}

	switch flags.ProviderName {
	case "gcp":
		return 1 * time.Hour, nil
	case "aws":
		return 15 * time.Minute, nil
	case "azure":
		return 1 * time.Hour, nil
	default:
		return 1 * time.Hour, nil
	}
}
