package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Long:  "Print detailed version information including build metadata",
		Run:   runVersion,
	}
}

func runVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("HyperFleet Cloud Provider\n")
	fmt.Printf("  Version:    %s\n", Version)
	fmt.Printf("  Commit:     %s\n", Commit)
	fmt.Printf("  Build Time: %s\n", BuildTime)
	fmt.Printf("  Go Version: %s\n", "go1.24+")
}
