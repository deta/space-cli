package cmd

import (
	"fmt"
	"strings"

	"github.com/deta/pc-cli/pkg/components/styles"
	"github.com/deta/pc-cli/shared"
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

func projectNotes(projectName string) string {
	return fmt.Sprintf(`
%s

👀 Find your project in Builder: %s
⚙️ Use %s the file to configure your app: %s
⚡ Push your code to Space with %s
🚀 Launch your app to the world with %s
`, styles.Bold("Next steps:"),
		styles.Bold(fmt.Sprintf("https://deta.space/builder/%s", projectName)),
		styles.Code("Space Manifest (space.yml)"), styles.Bold("https://alpha.deta.space/docs/en/reference/manifest"),
		styles.Code("deta push"), styles.Code("deta release"))
}
