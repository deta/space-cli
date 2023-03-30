package shared

import (
	"log"
	"os"

	"github.com/deta/space/internal/api"
)

const (
	DocsUrl          = "https://go.deta.dev/docs/space/alpha"
	SpacefileDocsUrl = "https://go.deta.dev/docs/spacefile/v0"
	BuilderUrl       = "https://deta.space/builder"
)

var (
	SpaceVersion string = "dev"
	Platform     string
	Client       = api.NewDetaClient(api.ClientConfig{Version: SpaceVersion, Platform: Platform})
	Logger       = log.New(os.Stderr, "", 0)
)
