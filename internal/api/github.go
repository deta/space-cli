package api

import (
	"context"
	"errors"
	"strings"

	"github.com/google/go-github/v51/github"
)

func findRelease(matchFunc func(*github.RepositoryRelease) bool) (*github.RepositoryRelease, error) {
	client := github.NewClient(nil)
	opt := &github.ListOptions{}
	for {
		releases, resp, err := client.Repositories.ListReleases(context.Background(), "deta", "space-cli", opt)
		if err != nil {
			return nil, err
		}

		for _, release := range releases {
			if matchFunc(release) {
				return release, nil
			}
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return nil, errors.New("release not found")
}

func GetLatestCliVersion() (string, error) {
	latestRelease, err := findRelease(func(release *github.RepositoryRelease) bool {
		return !release.GetPrerelease()
	})
	if err != nil {
		return "", err
	}

	tag := latestRelease.GetTagName()
	return strings.TrimPrefix(tag, "v"), nil
}
