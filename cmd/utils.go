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

func logMicros(micros []*shared.Micro) {
	logger.Println("Micros:")
	for _, micro := range micros {
		logMicro(micro)
	}
	logger.Println()
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
Notes:
	- Find your project in Builder here: https://deta.space/builder/%s
	- Your Space Manifest ("space.yml") contains the configuration of your project. 
	  Please modify your "space.yml" file to add your first Micro. 
	  Here is a reference: https://docs.deta.sh/manifest/add-micro
	- To push your code and create a Revision, use the command "deta push".
`, projectName)
}
