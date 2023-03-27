package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/deta/pc-cli/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type Page struct {
	Command      *cobra.Command
	HeaderOffset int
}

func main() {
	cmd := cmd.NewSpaceCmd()
	pages := []Page{
		{Command: cmd, HeaderOffset: -1},
	}

	commands := cmd.Commands()
	for _, command := range commands {
		pages = append(pages, Page{Command: command, HeaderOffset: 0})
	}

	out := bytes.Buffer{}
	for _, command := range pages {
		var page strings.Builder
		err := doc.GenMarkdown(command.Command, &page)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for _, line := range strings.Split(page.String(), "\n") {
			if strings.Contains(line, "SEE ALSO") {
				break
			}
			if strings.HasPrefix(line, "#") {
				if command.HeaderOffset < 0 {
					line = strings.TrimPrefix(line, strings.Repeat("#", -command.HeaderOffset))
				} else {
					line = strings.Repeat("#", command.HeaderOffset) + line
				}
			}

			out.WriteString(line + "\n")
		}
	}

	fmt.Println(out.String())
}
