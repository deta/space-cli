package shared

import (
	"log"
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
	Platform     string
	Client       = api.NewDetaClient(SpaceVersion, Platform)
	Logger       = log.New(os.Stderr, "", 0)
)
