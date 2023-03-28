package scanner

import (
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/deta/pc-cli/pkg/util/fs"
	"github.com/deta/pc-cli/shared"
)

var NodeFrameworks = [...]NodeFramework{
	{
		Name: shared.React,
		Detectors: Detectors{
			Matches: []Match{
				{Path: "package.json", MatchContent: `"(dev)?(d|D)ependencies":\s*{[^}]*"react-scripts":\s*".+?"[^}]*}`},
				{Path: "package.json", MatchContent: `"(dev)?(d|D)ependencies":\s*{[^}]*"react-dev-utils":\s*".+?"[^}]*}`},
			},
		},
	},
	{
		Name: shared.Svelte,
		Detectors: Detectors{
			Matches: []Match{
				{Path: "package.json", MatchContent: `"(dev)?(d|D)ependencies":\s*{[^}]*"svelte":\s*".+?"[^}]*}`},
				{Path: "package.json", MatchContent: `"(dev)?(d|D)ependencies":\s*{[^}]*"@sveltejs/vite-plugin-svelte":\s*".+?"[^}]*}`},
			},
			Strict: true,
		},
	},
	{
		Name: shared.Vue,
		Detectors: Detectors{
			Matches: []Match{
				{Path: "package.json", MatchContent: `"(dev)?(d|D)ependencies":\s*{[^}]*"@vue\/cli-service":\s*".+?"[^}]*}`},
			},
			Strict: true,
		},
	},
	{
		Name: shared.SvelteKit,
		Detectors: Detectors{
			Matches: []Match{
				{Path: "package.json", MatchContent: `"(dev)?(d|D)ependencies":\s*{[^}]*"@sveltejs\/kit":\s*".+?"[^}]*}`},
			},
			Strict: true,
		},
	},
	{
		Name: shared.Next,
		Detectors: Detectors{
			Matches: []Match{
				{Path: "package.json", MatchContent: `"(dev)?(d|D)ependencies":\s*{[^}]*"next":\s*".+?"[^}]*}`},
			},
			Strict: true,
		},
	},
	{
		Name: shared.Nuxt,
		Detectors: Detectors{
			Matches: []Match{
				{Path: "package.json", MatchContent: `"(dev)?(d|D)ependencies":\s*{[^}]*"nuxt3?(-edge)?":\s*".+?"[^}]*}`},
			},
			Strict: true,
		},
	},
}

func check(dir string, framework *NodeFramework) (bool, error) {
	passed := false
	for _, match := range (*framework).Detectors.Matches {

		// check to see if the file exists before checking for pattern match
		exists, err := fs.FileExists(dir, match.Path)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}

		path := filepath.Join(dir, match.Path)

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return false, err
		}

		pass, _ := regexp.MatchString(match.MatchContent, string(b))

		if !pass && (*framework).Detectors.Strict {
			return false, nil
		}
		if pass {
			passed = true
		}
	}
	return passed, nil
}

func detectFramework(dir string) (string, error) {
	for _, framework := range NodeFrameworks {
		check, err := check(dir, &framework)
		if err != nil {
			return "", err
		}
		if check {
			return framework.Name, nil
		}
	}
	return "nodejs16", nil
}
