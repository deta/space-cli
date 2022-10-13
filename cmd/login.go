package cmd

import (
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/spf13/cobra"
)

var (
	loginCmd = &cobra.Command{
		Use:   "login",
		Short: "login to space",
		RunE:  login,
	}
)

func init() {
	rootCmd.AddCommand(loginCmd)
}

func selectAccessToken() (string, error) {
	promptInput := text.Input{
		Prompt:      "Enter access token",
		Placeholder: "",
		Validator:   emptyPromptValidator,
	}

	return text.Run(&promptInput)
}

func login(cmd *cobra.Command, args []string) error {
	logger.Println()
	logger.Printf("To authenticate the Space CLI with your Space account, generate a new %s in your Space settings and paste it below:\n\n", styles.Code("access token"))
	accessToken, err := selectAccessToken()
	if err != nil {
		return err
	}

	err = auth.StoreAccessToken(accessToken)
	if err != nil {
		return err
	}

	logger.Println(styles.Green("👍 Login Successful!"))

	return nil
}
