package cmd

import (
	"github.com/spf13/cobra"
	"os"
)

var (
	rootCmd = &cobra.Command{
		Use:   "deta",
		Short: "Deta CLI for mananging deta micros",
		Long: `Deta command line interface for managing deta projects. 
Complete documentation available at https://docs.deta.sh`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
		// no usage shown on errors
		SilenceUsage: true,
	}
)

// Execute xx
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
