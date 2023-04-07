package shared

import (
	"fmt"
	"os"

	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/mattn/go-isatty"
)

func ProjectNotes(projectName string, projectId string) string {
	return fmt.Sprintf(`
%s

%s Find your project in Builder: %s
%s Use the %s to configure your app: %s
%s Push your code to Space with %s`, styles.Bold("Next steps:"), emoji.Eyes,
		styles.Bold(fmt.Sprintf("%s/%s", BuilderUrl, projectId)),
		emoji.Files,
		styles.Code("Spacefile"), styles.Bold(SpacefileDocsUrl),
		emoji.Swirl,
		styles.Code("space push"))
}

func LoginInfo() string {
	return styles.Boldf("No auth token found. Run %s or provide access token to login.", styles.Code("space login"))
}

func IsOutputInteractive() bool {
	return isatty.IsTerminal(os.Stdout.Fd())
}

func IsInputInteractive() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}
