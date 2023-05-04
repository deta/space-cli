package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/deta/space/pkg/util/fs"
	"github.com/deta/space/shared"
)

type engineScanner func(dir string) (*shared.Micro, error)

func pythonScanner(dir string) (*shared.Micro, error) {
	// if any of the following files exist detect as python app
	exists, err := fs.CheckIfAnyFileExists(dir, "requirements.txt", "Pipfile", "setup.py", "main.py")
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	name, err := getMicroNameFromPath(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract micro name from it's path, %v", err)
	}
	m := &shared.Micro{
		Name:   name,
		Src:    dir,
		Engine: shared.Python39,
	}

	return m, nil
}

type PackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (p *PackageJSON) HasDependency(pattern string) bool {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}

	for dep := range p.Dependencies {
		if reg.MatchString(dep) {
			return true
		}
	}

	for dep := range p.DevDependencies {
		if reg.MatchString(dep) {
			return true
		}
	}

	return false
}

func nodeScanner(dir string) (*shared.Micro, error) {
	name, err := getMicroNameFromPath(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract micro name from it's path, %v", err)
	}

	// if any of the following files exist detect as a node app
	manifestPath := filepath.Join(dir, "package.json")
	if _, err := os.Stat(manifestPath); err != nil {
		return nil, nil
	}

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var manifest PackageJSON
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, err
	}

	if manifest.HasDependency("react-scripts") || manifest.HasDependency("react-dev-utils") {
		return &shared.Micro{
			Name:   name,
			Src:    dir,
			Engine: shared.React,
		}, nil
	}

	if manifest.HasDependency("svelte") && manifest.HasDependency("@sveltejs/vite-plugin-svelte") {
		return &shared.Micro{
			Name:   name,
			Src:    dir,
			Engine: shared.Svelte,
		}, nil
	}

	if manifest.HasDependency("@vue/cli-service") {
		return &shared.Micro{
			Name:   name,
			Src:    dir,
			Engine: shared.Vue,
		}, nil
	}

	if manifest.HasDependency("@sveltejs/kit") {
		return &shared.Micro{
			Name:   name,
			Src:    dir,
			Engine: shared.SvelteKit,
		}, nil
	}

	if manifest.HasDependency("next") {
		return &shared.Micro{
			Name:   name,
			Src:    dir,
			Engine: shared.Next,
		}, nil
	}

	if manifest.HasDependency("nuxt3?(-edge)?") {
		return &shared.Micro{
			Name:   name,
			Src:    dir,
			Engine: shared.Nuxt,
		}, nil
	}

	return &shared.Micro{
		Name:   name,
		Src:    dir,
		Engine: "nodejs16",
	}, nil
}

func goScanner(dir string) (*shared.Micro, error) {
	name, err := getMicroNameFromPath(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract micro name from it's path, %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		return nil, nil
	}

	m := &shared.Micro{
		Name:     name,
		Src:      dir,
		Engine:   "custom",
		Commands: []string{"go build -o server"},
		Include:  []string{"server"},
		Run:      "./server",
	}

	return m, nil
}

func rustScanner(dir string) (*shared.Micro, error) {
	name, err := getMicroNameFromPath(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract micro name from it's path, %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err != nil {
		return nil, nil
	}

	return &shared.Micro{
		Name:     name,
		Src:      dir,
		Engine:   "custom",
		Commands: []string{"cargo build --release --bin server"},
		Include:  []string{"target/release/server"},
		Run:      "./server",
	}, nil
}

func staticScanner(dir string) (*shared.Micro, error) {
	// if any of the following files exist, detect as a static app
	exists, err := fs.CheckIfAnyFileExists(dir, "index.html")
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	name, err := getMicroNameFromPath(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to extract micro name from it's path, %v", err)
	}
	m := &shared.Micro{
		Name:   name,
		Src:    dir,
		Engine: shared.Static,
	}
	return m, nil
}
