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

func logScannedMicros(micros []*shared.Micro) {
	logger.Println("Scanned micros:")
	for _, micro := range micros {
		logMicro(micro)
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
