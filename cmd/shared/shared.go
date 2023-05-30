package shared

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/deta/space/internal/api"
)

const (
	DocsUrl          = "https://deta.space/docs"
	SpacefileDocsUrl = "https://deta.space/docs/en/reference/spacefile"
	BuilderUrl       = "https://deta.space/builder"
)

var (
	SpaceVersion string = "dev"
	DevPort      int    = 4200
	Platform     string
	Client       = api.NewDetaClient(SpaceVersion, Platform)
	Logger       = log.New(os.Stderr, "", 0)
)

func IsPortActive(port int) bool {
	conn, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}

	conn.Close()
	return true
}
