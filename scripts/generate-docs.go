package main

import (
	"fmt"
	"os"

	"github.com/deta/pc-cli/cmd"
)

func main() {
	doc, err := cmd.GenerateDocs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(doc)
}
