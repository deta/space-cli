package cmd

import (
	"fmt"
	"os"

	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/mattn/go-isatty"
)

const (
	docsUrl          = "https://go.deta.dev/docs/space/alpha"
	spacefileDocsUrl = "https://go.deta.dev/docs/spacefile/v0"
	builderUrl       = "https://deta.space/builder"
)

func projectNotes(projectName string, projectId string) string {
	return fmt.Sprintf(`
%s

%s Find your project in Builder: %s
%s Use the %s to configure your app: %s
%s Push your code to Space with %s`, styles.Bold("Next steps:"), emoji.Eyes,
		styles.Bold(fmt.Sprintf("%s/%s", builderUrl, projectId)),
		emoji.Files,
		styles.Code("Spacefile"), styles.Bold(spacefileDocsUrl),
		emoji.Swirl,
		styles.Code("space push"))
}

func LoginInfo() string {
	return styles.Boldf("No auth token found. Run %s or provide access token to login.", styles.Code("space login"))
}

func isOutputInteractive() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}
