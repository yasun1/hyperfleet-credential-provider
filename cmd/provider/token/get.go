package token

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/cmd/provider/common"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/execplugin"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/internal/provider"
	"github.com/openshift-hyperfleet/hyperfleet-cloud-provider/pkg/logger"
)

func NewCommand(flags *common.Flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-token",
		Short: "Generate a Kubernetes authentication token",
		Long: `Generate a short-lived authentication token for a Kubernetes cluster.

Outputs an ExecCredential JSON structure compatible with Kubernetes exec plugin.

Examples:
  # GCP/GKE
  hyperfleet-cloud-provider get-token --provider=gcp --cluster-name=my-cluster --project-id=my-project

  # AWS/EKS
  hyperfleet-cloud-provider get-token --provider=aws --cluster-name=my-cluster --region=us-east-1

  # Azure/AKS
  hyperfleet-cloud-provider get-token --provider=azure --cluster-name=my-cluster --tenant-id=... --subscription-id=...
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(flags)
		},
	}

	cmd.Flags().StringVar(&flags.ProviderName, "provider", "", "Cloud provider (gcp, aws, azure) [required]")
	cmd.Flags().StringVar(&flags.ClusterName, "cluster-name", "", "Cluster name [required]")
	cmd.Flags().StringVar(&flags.Region, "region", "", "Cloud region (optional for GCP, required for AWS, optional for Azure)")
	cmd.Flags().StringVar(&flags.ProjectID, "project-id", "", "GCP project ID (required for GCP)")
	cmd.Flags().StringVar(&flags.AccountID, "account-id", "", "AWS account ID (optional)")
	cmd.Flags().StringVar(&flags.SubscriptionID, "subscription-id", "", "Azure subscription ID (required for Azure)")
	cmd.Flags().StringVar(&flags.TenantID, "tenant-id", "", "Azure tenant ID (required for Azure)")

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

	log.Info("Starting token generation",
		logger.String("provider", flags.ProviderName),
		logger.String("cluster", flags.ClusterName),
	)

	prov, err := common.CreateProvider(flags, log)
	if err != nil {
		log.Error("Failed to create provider", logger.String("error", err.Error()))
		return err
	}

	opts := provider.GetTokenOptions{
		ClusterName:    flags.ClusterName,
		Region:         flags.Region,
		ProjectID:      flags.ProjectID,
		AccountID:      flags.AccountID,
		SubscriptionID: flags.SubscriptionID,
		TenantID:       flags.TenantID,
	}

	token, err := prov.GetToken(ctx, opts)
	if err != nil {
		log.Error("Failed to generate token", logger.String("error", err.Error()))
		return err
	}

	log.Info("Token generated successfully",
		logger.String("provider", flags.ProviderName),
		logger.String("expires_at", token.ExpiresAt.Format(time.RFC3339)),
	)

	writer := execplugin.NewOutputWriter(os.Stdout)
	if err := writer.WriteToken(token); err != nil {
		log.Error("Failed to write token output", logger.String("error", err.Error()))
		return err
	}

	return nil
}
