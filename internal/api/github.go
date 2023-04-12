package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v51/github"
)

func GetLatestCliVersion() (string, error) {
	client := github.NewClient(nil)

	release, resp, err := client.Repositories.GetLatestRelease(context.Background(), "deta", "space-cli")

	if err != nil {
		return "", fmt.Errorf("error while fetching latest release: %v", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("error while fetching latest release: %v", resp.Status)
	}

	return strings.TrimPrefix(release.GetTagName(), "v"), nil
}
