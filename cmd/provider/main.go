package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/cluster"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/common"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/kubeconfig"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/token"
	"github.com/openshift-hyperfleet/hyperfleet-credential-provider/cmd/provider/version"
)

func main() {
	flags := &common.Flags{}

	rootCmd := &cobra.Command{
		Use:   "hyperfleet-credential-provider",
		Short: "Multi-cloud Kubernetes authentication token provider",
		Long: `HyperFleet Credential Provider generates short-lived Kubernetes authentication tokens
for GKE, EKS, and AKS clusters without requiring cloud CLIs.

Supports Kubernetes exec plugin authentication for seamless cluster access.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVar(&flags.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&flags.LogFormat, "log-format", "json", "Log format (json, console)")
	rootCmd.PersistentFlags().StringVar(&flags.CredentialsFile, "credentials-file", "", "Path to credentials file (overrides environment variables)")

	// Initialize Viper for environment variable support
	cobra.OnInitialize(common.InitViper)

	// Bind persistent flags to viper (global flags available to all subcommands)
	common.BindPersistentFlags(rootCmd)

	rootCmd.AddCommand(version.NewCommand())
	rootCmd.AddCommand(token.NewCommand(flags))
	rootCmd.AddCommand(cluster.NewCommand(flags))
	rootCmd.AddCommand(kubeconfig.NewCommand(flags))

	// Execute
	if err := rootCmd.Execute(); err != nil {
		// Print error to stderr since we have SilenceErrors: true
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
