// Command swm is the Story-based Workflow Manager host.
// Phase 0: just the root command with --version. Real subcommands in Phase 1+.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "v2.0.0-dev"

func main() {
	root := &cobra.Command{
		Use:     "swm",
		Short:   "Story-based Workflow Manager",
		Version: version,
		// SilenceUsage prevents cobra from printing usage on every error.
		SilenceUsage: true,
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
