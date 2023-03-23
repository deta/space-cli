package cmd

import (
	"fmt"
	"strings"

	"github.com/deta/pc-cli/pkg/components/emoji"
	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/shared"
)

const (
	docsUrl          = "https://go.deta.dev/docs/space/alpha"
	spacefileDocsUrl = "https://go.deta.dev/docs/spacefile/v0"
	builderUrl       = "https://deta.space/builder"
)

func isFlagEmpty(flag string) bool {
	return strings.TrimSpace(flag) == ""
}

func logDetectedMicros(micros []*shared.Micro) {
	for _, micro := range micros {
		logger.Printf("Micro found in \"%s\"\n", styles.Code(fmt.Sprintf("%s/", micro.Src)))
		logger.Printf("L engine: %s\n\n", styles.Blue(micro.Engine))
	}
}

func emptyPromptValidator(value string) error {
	if value == "" {
		return fmt.Errorf("cannot be empty")
	}
	return nil
}

func projectIDValidator(projectID string) error {
	if projectID == "" {
		return fmt.Errorf("please provide a valid id, empty project id is not valid")
	}
	return nil
}

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
