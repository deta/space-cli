package cmd

import (
	"errors"
	"io"
	"os"

	"github.com/deta/pc-cli/internal/api"
	"github.com/deta/pc-cli/internal/auth"
	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/pkg/components/text"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	loginCmd = &cobra.Command{
		Use:   "login",
		Short: "login to space",
		RunE:  login,
		Example: `# start interactive login
space login

# login by reading token from a file
space login --with-token < token.txt`,
	}
)

func init() {
	loginCmd.Flags().Bool("with-token", false, "Read token from standard input")
	rootCmd.AddCommand(loginCmd)
}

func selectAccessToken() (string, error) {
	promptInput := text.Input{
		Prompt:       "Enter access token",
		Placeholder:  "",
		Validator:    emptyPromptValidator,
		PasswordMode: true,
	}

	return text.Run(&promptInput)
}

func isOutputInteractive() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

func login(cmd *cobra.Command, args []string) (err error) {
	logger.Println()

	// check space version
	c := make(chan *checkVersionMsg, 1)
	defer close(c)
	go checkVersion(c)

	var accessToken string
	withToken, _ := cmd.Flags().GetBool("with-token")
	if withToken {
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			return errors.New("Failed to read token from standard input")
		}

		accessToken = string(input)
	} else {
		if !isOutputInteractive() {
			return errors.New("Cannot start interactive login when standard input is not a terminal")
		}

		logger.Printf("To authenticate the Space CLI with your Space account, generate a new %s in your Space settings and paste it below:\n\n", styles.Code("access token"))
		accessToken, err = selectAccessToken()
		if err != nil {
			return err
		}
	}

	_, err = client.GetSpace(&api.GetSpaceRequest{
		AccessToken: accessToken,
	})
	if err != nil {
		if errors.Is(err, auth.ErrInvalidAccessToken) {
			logger.Printf(styles.Errorf("%s Invalid access token. Please generate a valid token from your Space settings.", emoji.ErrorExclamation))
			return nil
		}
		logger.Printf(styles.Errorf("%s Failed to validate access token: %v", emoji.ErrorExclamation, err))
		return nil
	}

	if err = auth.StoreAccessToken(accessToken); err != nil {
		return err
	}

	logger.Println(styles.Green("👍 Login Successful!"))
	cm := <-c
	if cm.err == nil && cm.isLower {
		logger.Println(styles.Boldf("\n%s New Space CLI version available, upgrade with %s", emoji.Rocket, styles.Code("space version upgrade")))
	}
	return nil
}
