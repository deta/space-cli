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
