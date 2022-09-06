package cmd

import (
	"log"
	"os"

	"github.com/deta/pc-cli/internal/api"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "deta",
		Short: "Deta CLI for mananging deta space projects",
		Long: `Deta command line interface for managing space projects. 
Complete documentation available at https://docs.deta.sh`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
		// no usage shown on errors
		SilenceUsage: true,
	}

	client = api.NewDetaClient()

	logger = log.New(os.Stderr, "", 0)
)

// Execute xx
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
