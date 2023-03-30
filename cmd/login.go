package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/deta/space/cmd/shared"
	"github.com/deta/space/internal/api"
	"github.com/deta/space/internal/auth"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/deta/space/pkg/components/text"
	"github.com/spf13/cobra"
)

func newCmdLogin() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "login to space",
		Run: func(cmd *cobra.Command, args []string) {
			var err error
			withToken, _ := cmd.Flags().GetBool("with-token")

			var accessToken string
			if withToken {
				input, err := io.ReadAll(os.Stdin)
				if err != nil {
					shared.Logger.Println("failed to read access token from standard input")
					os.Exit(1)
				}

				accessToken = strings.TrimSpace(string(input))
			} else {
				shared.Logger.Printf("To authenticate the Space CLI with your Space account, generate a new %s in your Space settings and paste it below:\n\n", styles.Code("access token"))
				accessToken, err = inputAccessToken()
				if err != nil {
					os.Exit(1)
				}
			}

			if err := login(accessToken); err != nil {
				shared.Logger.Printf(styles.Errorf("%s Failed to login: %v", emoji.ErrorExclamation, err))
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolP("with-token", "t", false, "Read token from standard input")
	if !shared.IsOutputInteractive() {
		cmd.MarkFlagRequired("with-token")
	}

	return cmd
}

func inputAccessToken() (string, error) {
	promptInput := text.Input{
		Prompt:      "Enter access token",
		Placeholder: "",
		Validator: func(value string) error {
			if value == "" {
				return fmt.Errorf("cannot be empty")
			}
			return nil
		},
		PasswordMode: true,
	}

	return text.Run(&promptInput)
}

func login(accessToken string) (err error) {
	// Check if the access token is valid
	_, err = shared.Client.GetSpace(&api.GetSpaceRequest{
		AccessToken: accessToken,
	})

	if err != nil {
		if errors.Is(err, auth.ErrInvalidAccessToken) {
			shared.Logger.Printf(styles.Errorf("%s Invalid access token. Please generate a valid token from your Space settings.", emoji.ErrorExclamation))
			return fmt.Errorf("invalid access token")
		}
		shared.Logger.Printf(styles.Errorf("%s Failed to validate access token: %v", emoji.ErrorExclamation, err))
		return fmt.Errorf("failed to validate access token: %w", err)
	}

	err = auth.StoreAccessToken(accessToken)
	if err != nil {
		return fmt.Errorf("failed to store access token: %w", err)
	}

	shared.Logger.Println(styles.Green("👍 Login Successful!"))
	return nil
}
