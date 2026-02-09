package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/common"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/aws"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/azure"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/gcp"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

func NewCommand(flags *common.Flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-cluster-info",
		Short: "Get cluster information (endpoint, CA certificate)",
		Long: `Get cluster information including API server endpoint and CA certificate.

This command uses cloud provider SDKs (no CLI required) to fetch cluster details.

Examples:
  # GCP/GKE
  hyperfleet-credential-provider get-cluster-info \
    --provider=gcp \
    --cluster-name=my-cluster \
    --project-id=my-project \
    --region=us-central1

  # AWS/EKS
  hyperfleet-credential-provider get-cluster-info \
    --provider=aws \
    --cluster-name=my-cluster \
    --region=us-east-1

  # Azure/AKS
  hyperfleet-credential-provider get-cluster-info \
    --provider=azure \
    --cluster-name=my-cluster \
    --subscription-id=xxx \
    --tenant-id=xxx \
    --resource-group=my-rg

  # Output example:
  {
    "endpoint": "https://34.68.222.124",
    "certificateAuthority": "LS0tLS1CRUdJTi...",
    "version": "v1.33.5-gke.2118001"
  }`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(flags)
		},
	}

	cmd.Flags().StringVar(&flags.ProviderName, "provider", "", "Cloud provider (gcp, aws, azure) [required]")
	cmd.Flags().StringVar(&flags.ClusterName, "cluster-name", "", "Cluster name [required]")
	cmd.Flags().StringVar(&flags.Region, "region", "", "Cloud region/location [required for GCP/AWS]")
	cmd.Flags().StringVar(&flags.ProjectID, "project-id", "", "GCP project ID (required for GCP)")
	cmd.Flags().StringVar(&flags.AccountID, "account-id", "", "AWS account ID (optional)")
	cmd.Flags().StringVar(&flags.SubscriptionID, "subscription-id", "", "Azure subscription ID (required for Azure)")
	cmd.Flags().StringVar(&flags.TenantID, "tenant-id", "", "Azure tenant ID (required for Azure)")
	cmd.Flags().StringVar(&flags.ResourceGroup, "resource-group", "", "Azure resource group (required for Azure)")

	// Bind flags to viper for environment variable support
	common.BindCommandFlags(cmd)

	// Note: We don't use MarkFlagRequired because Cobra validates before Viper bindings take effect
	// Instead, we validate in the run function after BindFlagsToViper is called

	return cmd
}

func run(flags *common.Flags) error {
	// Bind Viper values to flags (environment variables take precedence if flags not set)
	common.BindFlagsToViper(flags)

	if flags.ProviderName == "" {
		return fmt.Errorf("--provider is required (or set HFCP_PROVIDER)")
	}
	if flags.ClusterName == "" {
		return fmt.Errorf("--cluster-name is required (or set HFCP_CLUSTER_NAME)")
	}

	log, err := logger.New(logger.Config{
		Level:  logger.Level(flags.LogLevel),
		Format: logger.Format(flags.LogFormat),
	})
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer log.Sync()

	ctx := context.Background()

	log.Info("Fetching cluster information",
		logger.String("provider", flags.ProviderName),
		logger.String("cluster", flags.ClusterName),
	)

	switch flags.ProviderName {
	case "gcp":
		return getGCPClusterInfo(ctx, flags, log)
	case "aws":
		return getAWSClusterInfo(ctx, flags, log)
	case "azure":
		return getAzureClusterInfo(ctx, flags, log)
	default:
		return fmt.Errorf("unsupported provider: %s (must be one of: gcp, aws, azure)", flags.ProviderName)
	}
}

func getGCPClusterInfo(ctx context.Context, flags *common.Flags, log logger.Logger) error {
	if flags.ProjectID == "" {
		return fmt.Errorf("--project-id is required for GCP")
	}
	if flags.Region == "" {
		return fmt.Errorf("--region is required for GCP (location can be region or zone)")
	}

	config := &gcp.Config{
		ProjectID:       flags.ProjectID,
		CredentialsFile: flags.CredentialsFile,
		TokenDuration:   1 * time.Hour,
	}
	provider, err := gcp.NewProvider(config, log)
	if err != nil {
		return fmt.Errorf("failed to create GCP provider: %w", err)
	}

	info, err := provider.GetClusterInfo(ctx, flags.ClusterName, flags.Region)
	if err != nil {
		return fmt.Errorf("failed to get cluster info: %w", err)
	}

	output := map[string]string{
		"endpoint":             "https://" + info.Endpoint,
		"certificateAuthority": info.CertificateAuthority,
		"version":              info.Version,
		"location":             info.Location,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	return nil
}

func getAWSClusterInfo(ctx context.Context, flags *common.Flags, log logger.Logger) error {
	if flags.Region == "" {
		return fmt.Errorf("--region is required for AWS")
	}

	config := &aws.Config{
		Region:          flags.Region,
		CredentialsFile: flags.CredentialsFile,
		TokenDuration:   15 * time.Minute,
	}
	provider, err := aws.NewProvider(config, log)
	if err != nil {
		return fmt.Errorf("failed to create AWS provider: %w", err)
	}

	info, err := provider.GetClusterInfo(ctx, flags.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to get cluster info: %w", err)
	}

	output := map[string]string{
		"endpoint":             info.Endpoint,
		"certificateAuthority": info.CertificateAuthority,
		"version":              info.Version,
		"region":               info.Region,
		"arn":                  info.ARN,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	return nil
}

func getAzureClusterInfo(ctx context.Context, flags *common.Flags, log logger.Logger) error {
	if flags.SubscriptionID == "" {
		return fmt.Errorf("--subscription-id is required for Azure")
	}
	if flags.TenantID == "" {
		return fmt.Errorf("--tenant-id is required for Azure")
	}
	if flags.ResourceGroup == "" {
		return fmt.Errorf("--resource-group is required for Azure")
	}

	config := &azure.Config{
		TenantID:        flags.TenantID,
		SubscriptionID:  flags.SubscriptionID,
		CredentialsFile: flags.CredentialsFile,
		TokenDuration:   1 * time.Hour,
	}
	provider, err := azure.NewProvider(config, log)
	if err != nil {
		return fmt.Errorf("failed to create Azure provider: %w", err)
	}

	info, err := provider.GetClusterInfo(ctx, flags.ClusterName, flags.ResourceGroup)
	if err != nil {
		return fmt.Errorf("failed to get cluster info: %w", err)
	}

	output := map[string]string{
		"endpoint":             info.Endpoint,
		"certificateAuthority": info.CertificateAuthority,
		"version":              info.Version,
		"location":             info.Location,
		"resourceId":           info.ResourceID,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("failed to encode output: %w", err)
	}

	return nil
}
