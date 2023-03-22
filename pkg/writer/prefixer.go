package writer

import (
	"fmt"
	"io"
	"strings"
)

type Prefixer struct {
	scope string
	dest  io.Writer
}

func NewPrefixer(scope string, dest io.Writer) *Prefixer {
	return &Prefixer{
		scope: scope,
		dest:  dest,
	}
}

// parse the logs and prefix them with the scope
func (p Prefixer) Write(bytes []byte) (int, error) {
	normalized := strings.ReplaceAll(string(bytes), "\r\n", "\n")
	lines := strings.Split(normalized, "\n")

	for _, line := range lines {
		fmt.Printf("[%s] %s\n", p.scope, line)
	}

	return len(bytes), nil
}
