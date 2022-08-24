package cmd

import (
	"log"

	"github.com/deta/pc-cli/pkg/choose"
	"github.com/deta/pc-cli/pkg/confirm"
	"github.com/deta/pc-cli/pkg/text"
	"github.com/spf13/cobra"
)

var (
	name        string
	dir         string
	confirmFlag bool

	newCmd = &cobra.Command{
		Use:   "new [flags]",
		Short: "Create a new project",
		RunE:  new,
	}
)

func init() {
	newCmd.Flags().StringVarP(&name, "name", "n", "default", "name of the new project")
	newCmd.Flags().StringVarP(&dir, "dir", "d", ".", "where the project is created")
	newCmd.Flags().BoolVarP(&confirmFlag, "confirm", "c", false, "prefill missing arguments")

	rootCmd.AddCommand(newCmd)
}

func new(cmd *cobra.Command, args []string) error {
	_, err := text.Run(&text.Input{Prompt: "What's your app's name?", Placeholder: "default"})
	if err != nil {
		log.Println("Error:", err)
	}

	_, err = choose.Run(&choose.Input{Prompt: "What type of micro do you want to create?", Choices: []string{"static", "fullstack", "native", "custom"}})
	if err != nil {
		log.Println("Error:", err)
	}

	_, err = confirm.Run(&confirm.Input{Prompt: "Do you want to continue?"})
	if err != nil {
		log.Println("Error:", err)
	}

	return nil
}
