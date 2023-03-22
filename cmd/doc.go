package cmd

import (
	"bytes"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func GenerateDocs() (string, error) {

	out := bytes.Buffer{}
	pages := []struct {
		command      *cobra.Command
		headerOffset int
	}{
		{command: rootCmd, headerOffset: 0},
		{command: newCmd, headerOffset: 1},
		{command: linkCmd, headerOffset: 1},
		{command: validateCmd, headerOffset: 1},
		{command: devCmd, headerOffset: 1},
		{command: devTriggerCmd, headerOffset: 2},
		{command: devUpCmd, headerOffset: 2},
		{command: devProxyCmd, headerOffset: 2},
		{command: pushCmd, headerOffset: 1},
		{command: openCmd, headerOffset: 1},
		{command: releaseCmd, headerOffset: 1},
		{command: versionCmd, headerOffset: 1},
	}

	for _, command := range pages {
		var page strings.Builder
		err := doc.GenMarkdown(command.command, &page)
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(page.String(), "\n") {
			if strings.Contains(line, "SEE ALSO") {
				break
			}
			if strings.HasPrefix(line, "#") {
				line = strings.Repeat("#", command.headerOffset) + line
			}

			out.WriteString(line + "\n")
		}
	}

	return out.String(), nil
}
