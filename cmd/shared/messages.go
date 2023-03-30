package shared

import (
	"fmt"
	"log"
	"os"

	"github.com/deta/space/internal/api"
	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/mattn/go-isatty"
)

const (
	DocsUrl          = "https://go.deta.dev/docs/space/alpha"
	SpacefileDocsUrl = "https://go.deta.dev/docs/spacefile/v0"
	BuilderUrl       = "https://deta.space/builder"
)

var (
	Client = api.NewDetaClient({ version: spaceVersion, platform: platform })
	Logger = log.New(os.Stderr, "", 0)
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
