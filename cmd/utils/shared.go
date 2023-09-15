package utils

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/deta/space/internal/api"
)

const (
	DocsUrl          = "https://deta.space/docs"
	SpacefileDocsUrl = "https://go.deta.dev/docs/spacefile/v0"
	BuilderUrl       = "https://deta.space/builder"
)

var (
	Platform     string
	SpaceVersion = "dev"
	DevPort      = 4200
	Client       = api.NewDetaClient(SpaceVersion, Platform)
	Logger       = log.New(os.Stdout, "", 0)
	StdErrLogger = log.New(os.Stderr, "", 0)
)

func IsPortActive(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}

	conn.Close()
	return true
}
