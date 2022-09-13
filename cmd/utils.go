package cmd

import (
	"fmt"
	"strings"

	"github.com/deta/pc-cli/shared"
)

func isFlagEmpty(flag string) bool {
	return strings.TrimSpace(flag) == ""
}

func logMicro(micro *shared.Micro) {
	msg := fmt.Sprintf("name: %s\n", micro.Name)
	msg += fmt.Sprintf(" L src: %s\n", micro.Src)
	msg += fmt.Sprintf(" L engine: %s", micro.Engine)
	logger.Println(msg)
}

func logDetectedMicros(micros []*shared.Micro) {
	for _, micro := range micros {
		logger.Printf("Micro found in \"%s/\"\n", micro.Src)
		logger.Printf("L engine: %s\n\n", micro.Engine)
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
Next Steps:

👀 Find your project in Builder: https://deta.space/builder/%s
⚙️ Use the "space.yml" file to configure your app: https://alpha.deta.space/docs/en/reference/manifest
⚡ Push your code to Space with "deta push"
🚀 Launch your app to the world with "deta release"
`, projectName)
}
