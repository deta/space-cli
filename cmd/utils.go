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
	builderUrl       = "https://alpha.deta.space/builder"
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
%s Push your code to Space with %s
%s Launch your app to the world with %s`, styles.Bold("Next steps:"), emoji.Eyes,
		styles.Bold(fmt.Sprintf("%s/%s", builderUrl, projectId)),
		emoji.Files,
		styles.Code("Spacefile"), styles.Bold(spacefileDocsUrl),
		emoji.Swirl,
		styles.Code("space push"),
		emoji.Rocket,
		styles.Code("space release"))
}

func SpaceGitignoreInfo() string {
	return styles.Boldf("%s Make sure to not commit %s to git as it may contain security-sensitive data. Make sure to add it to your %s", styles.Info, styles.Code(".space"), styles.Code(".gitignore"))
}
