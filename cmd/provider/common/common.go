package common

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider/aws"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider/azure"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider/gcp"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
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

	// Create logger with stderr output (stdout is reserved for ExecCredential)
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
