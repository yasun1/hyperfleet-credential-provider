package kubeconfig

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/common"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/aws"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/azure"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/internal/provider/gcp"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/pkg/logger"
)

var outputFile string

func NewCommand(flags *common.Flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-kubeconfig",
		Short: "Generate a complete kubeconfig file",
		Long: `Generate a complete kubeconfig file for the specified cluster.

This command uses cloud provider SDKs (no CLI required) to fetch cluster details
and generates a kubeconfig that uses hyperfleet-credential-provider for token generation.

Examples:
  # GCP/GKE
  hyperfleet-credential-provider generate-kubeconfig \
    --provider=gcp \
    --cluster-name=my-cluster \
    --project-id=my-project \
    --region=us-central1 \
    --output=kubeconfig.yaml`,
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
	cmd.Flags().StringVar(&outputFile, "output", "", "Output file path (default: stdout)")
	cmd.Flags().StringVar(&flags.TokenDuration, "token-duration", "", "Token duration (e.g., 1h, 30m, 900s) (default: GCP=1h, AWS=15m, Azure=1h)")

	cmd.MarkFlagRequired("provider")
	cmd.MarkFlagRequired("cluster-name")

	return cmd
}

func run(flags *common.Flags) error {
	ctx, cancel := common.SetupSignalHandler()
	defer cancel()

	log, err := common.CreateLogger(flags)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer log.Sync()

	log.Info("Generating kubeconfig",
		logger.String("provider", flags.ProviderName),
		logger.String("cluster", flags.ClusterName),
	)

	var endpoint, caCert, version string
	var providerSpecificInfo map[string]string

	switch flags.ProviderName {
	case "gcp":
		endpoint, caCert, version, providerSpecificInfo, err = getGCPClusterInfoForKubeconfig(ctx, flags, log)
	case "aws":
		endpoint, caCert, version, providerSpecificInfo, err = getAWSClusterInfoForKubeconfig(ctx, flags, log)
	case "azure":
		endpoint, caCert, version, providerSpecificInfo, err = getAzureClusterInfoForKubeconfig(ctx, flags, log)
	default:
		return fmt.Errorf("unsupported provider: %s (must be gcp, aws, or azure)", flags.ProviderName)
	}

	if err != nil {
		return fmt.Errorf("failed to get cluster info: %w", err)
	}

	log.Info("Cluster info retrieved",
		logger.String("endpoint", endpoint),
		logger.String("version", version),
	)

	kubeconfig, err := generateKubeconfigYAML(endpoint, caCert, providerSpecificInfo)
	if err != nil {
		return fmt.Errorf("failed to generate kubeconfig: %w", err)
	}

	if outputFile != "" {
		if err := os.WriteFile(outputFile, kubeconfig, 0600); err != nil {
			return fmt.Errorf("failed to write kubeconfig to file: %w", err)
		}
		log.Info("Kubeconfig written to file",
			logger.String("file", outputFile),
		)
		fmt.Fprintf(os.Stderr, "✅ Kubeconfig generated: %s\n", outputFile)
	} else {
		fmt.Print(string(kubeconfig))
	}

	return nil
}

func getGCPClusterInfoForKubeconfig(ctx context.Context, flags *common.Flags, log logger.Logger) (string, string, string, map[string]string, error) {
	if flags.ProjectID == "" {
		return "", "", "", nil, fmt.Errorf("--project-id is required for GCP")
	}
	if flags.Region == "" {
		return "", "", "", nil, fmt.Errorf("--region is required for GCP (location can be region or zone)")
	}

	duration, err := common.ParseTokenDuration(flags)
	if err != nil {
		return "", "", "", nil, err
	}

	config := &gcp.Config{
		ProjectID:       flags.ProjectID,
		CredentialsFile: flags.CredentialsFile,
		TokenDuration:   duration,
	}
	provider, err := gcp.NewProvider(config, log)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("failed to create GCP provider: %w", err)
	}

	info, err := provider.GetClusterInfo(ctx, flags.ClusterName, flags.Region)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	endpoint := "https://" + info.Endpoint
	providerInfo := map[string]string{
		"provider":     "gcp",
		"cluster-name": flags.ClusterName,
		"project-id":   flags.ProjectID,
		"region":       flags.Region,
		"creds-env":    "GOOGLE_APPLICATION_CREDENTIALS",
		"creds-path":   common.GetCredentialsPath(flags),
	}

	return endpoint, info.CertificateAuthority, info.Version, providerInfo, nil
}

func getAWSClusterInfoForKubeconfig(ctx context.Context, flags *common.Flags, log logger.Logger) (string, string, string, map[string]string, error) {
	if flags.Region == "" {
		return "", "", "", nil, fmt.Errorf("--region is required for AWS")
	}

	duration, err := common.ParseTokenDuration(flags)
	if err != nil {
		return "", "", "", nil, err
	}

	config := &aws.Config{
		Region:          flags.Region,
		CredentialsFile: flags.CredentialsFile,
		TokenDuration:   duration,
	}
	provider, err := aws.NewProvider(config, log)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("failed to create AWS provider: %w", err)
	}

	info, err := provider.GetClusterInfo(ctx, flags.ClusterName)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	providerInfo := map[string]string{
		"provider":     "aws",
		"cluster-name": flags.ClusterName,
		"region":       flags.Region,
		"creds-env":    "AWS_CREDENTIALS_FILE",
		"creds-path":   common.GetCredentialsPath(flags),
	}

	return info.Endpoint, info.CertificateAuthority, info.Version, providerInfo, nil
}

func getAzureClusterInfoForKubeconfig(ctx context.Context, flags *common.Flags, log logger.Logger) (string, string, string, map[string]string, error) {
	if flags.SubscriptionID == "" {
		return "", "", "", nil, fmt.Errorf("--subscription-id is required for Azure")
	}
	if flags.TenantID == "" {
		return "", "", "", nil, fmt.Errorf("--tenant-id is required for Azure")
	}
	if flags.ResourceGroup == "" {
		return "", "", "", nil, fmt.Errorf("--resource-group is required for Azure")
	}

	duration, err := common.ParseTokenDuration(flags)
	if err != nil {
		return "", "", "", nil, err
	}

	config := &azure.Config{
		SubscriptionID:  flags.SubscriptionID,
		TenantID:        flags.TenantID,
		CredentialsFile: flags.CredentialsFile,
		TokenDuration:   duration,
	}
	provider, err := azure.NewProvider(config, log)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("failed to create Azure provider: %w", err)
	}

	info, err := provider.GetClusterInfo(ctx, flags.ClusterName, flags.ResourceGroup)
	if err != nil {
		return "", "", "", nil, fmt.Errorf("failed to get cluster info: %w", err)
	}

	providerInfo := map[string]string{
		"provider":        "azure",
		"cluster-name":    flags.ClusterName,
		"subscription-id": flags.SubscriptionID,
		"tenant-id":       flags.TenantID,
		"resource-group":  flags.ResourceGroup,
		"creds-env":       "AZURE_CREDENTIALS_FILE",
		"creds-path":      common.GetCredentialsPath(flags),
	}

	return info.Endpoint, info.CertificateAuthority, info.Version, providerInfo, nil
}

func generateKubeconfigYAML(endpoint, caCert string, providerInfo map[string]string) ([]byte, error) {
	clusterName := providerInfo["cluster-name"]
	userName := "hyperfleet-user"
	contextName := clusterName

	execArgs := []string{"get-token", "--provider=" + providerInfo["provider"], "--cluster-name=" + clusterName}

	switch providerInfo["provider"] {
	case "gcp":
		execArgs = append(execArgs, "--project-id="+providerInfo["project-id"])
		execArgs = append(execArgs, "--region="+providerInfo["region"])
	case "aws":
		execArgs = append(execArgs, "--region="+providerInfo["region"])
	case "azure":
		execArgs = append(execArgs, "--subscription-id="+providerInfo["subscription-id"])
		execArgs = append(execArgs, "--tenant-id="+providerInfo["tenant-id"])
	}

	kubeconfig := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Config",
		"clusters": []map[string]interface{}{
			{
				"name": clusterName,
				"cluster": map[string]interface{}{
					"server":                     endpoint,
					"certificate-authority-data": caCert,
				},
			},
		},
		"users": []map[string]interface{}{
			{
				"name": userName,
				"user": map[string]interface{}{
					"exec": map[string]interface{}{
						"apiVersion": "client.authentication.k8s.io/v1",
						"command":    "hyperfleet-credential-provider",
						"args":       execArgs,
						"env": []map[string]string{
							{
								"name":  providerInfo["creds-env"],
								"value": providerInfo["creds-path"],
							},
						},
						"interactiveMode": "Never",
					},
				},
			},
		},
		"contexts": []map[string]interface{}{
			{
				"name": contextName,
				"context": map[string]interface{}{
					"cluster": clusterName,
					"user":    userName,
				},
			},
		},
		"current-context": contextName,
	}

	yamlData, err := yaml.Marshal(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubeconfig to YAML: %w", err)
	}

	return yamlData, nil
}
