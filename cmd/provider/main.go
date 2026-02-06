package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/cluster"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/common"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/kubeconfig"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/token"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/version"
)

func main() {
	// Create shared flags struct
	flags := &common.Flags{}

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "hyperfleet-credential-provider",
		Short: "Multi-cloud Kubernetes authentication token provider",
		Long: `HyperFleet Credential Provider generates short-lived Kubernetes authentication tokens
for GKE, EKS, and AKS clusters without requiring cloud CLIs.

Supports Kubernetes exec plugin authentication for seamless cluster access.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Add global flags
	rootCmd.PersistentFlags().StringVar(&flags.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&flags.LogFormat, "log-format", "json", "Log format (json, console)")
	rootCmd.PersistentFlags().StringVar(&flags.CredentialsFile, "credentials-file", "", "Path to credentials file (overrides environment variables)")

	// Add subcommands
	rootCmd.AddCommand(version.NewCommand())
	rootCmd.AddCommand(token.NewCommand(flags))
	rootCmd.AddCommand(cluster.NewCommand(flags))
	rootCmd.AddCommand(kubeconfig.NewCommand(flags))

	// Execute
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
