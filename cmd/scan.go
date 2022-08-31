package cmd

import (
	"os"
	"path/filepath"

	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/deta/pc-cli/pkg/scanner"
	"github.com/spf13/cobra"
)

var (
	dir string

	scanCmd = &cobra.Command{
		Use:   "scan [flags]",
		Short: "scan micros in dir",
		RunE:  scan,
	}
)

func init() {
	scanCmd.Flags().StringVarP(&dir, "dir", "d", "", "where should the scanner run?")
	rootCmd.AddCommand(scanCmd)
}

func isFlagEmpty(flag string) bool {
	return flag == ""
}

func selectDir() (string, error) {
	promptInput := text.Input{
		Prompt:      "Where do you want to scan?",
		Placeholder: "./",
	}

	return text.Run(&promptInput)
}

func scan(cmd *cobra.Command, args []string) error {
	var err error

	// prompt user for dir if not provided via args
	if isFlagEmpty(dir) {
		dir, err = selectDir()
		if err != nil {
			logger.Printf("problem while trying to retrieve dir to scan through text prompt, %v\n", err)
			return nil
		}
	}

	// scan dir for micros
	micros, err := scanner.Scan(dir)
	if err != nil {
		logger.Printf("failed to scan micros, %v\n", err)
		return nil
	}

	wd, err := os.Getwd()
	if err != nil {
		logger.Printf("failed to read parent folder's name for project name\n")
	}
	project := filepath.Base(wd)

	logger.Println("Scanned micros:")
	logger.Printf("%s\n", project)
	for _, micro := range micros {
		logger.Printf("L%s\n Src - %s\n Engine - %s", micro.Name, micro.Src, micro.Engine)
	}

	return nil
}
