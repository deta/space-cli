package scanner

import (
	"fmt"

	"github.com/deta/pc-cli/pkg/util/fs"
	"github.com/deta/pc-cli/types"
)

func pythonScanner(dir string) (*types.Micro, error) {
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
	m := &types.Micro{
		Name:   name,
		Src:    dir,
		Engine: types.Python39,
	}

	return m, nil
}

func nodeScanner(dir string) (*types.Micro, error) {
	// if any of the following files exist detect as a node app
	exists, err := fs.CheckIfAnyFileExists(dir, "package.json")
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

	m := &types.Micro{
		Name:   name,
		Src:    dir,
		Engine: types.Node16x,
	}

	framework, err := detectFramework(dir)
	if err != nil {
		return nil, err
	}
	m.Engine = framework

	return m, nil
}

func goScanner(dir string) (*types.Micro, error) {
	exists, err := fs.CheckIfAnyFileExists(dir, "go.mod")
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
	m := &types.Micro{
		Name:     name,
		Src:      dir,
		Engine:   "custom",
		Commands: []string{"go build cmd/main.go"},
		Include:  []string{"main"},
		Run:      "./main",
	}

	return m, nil
}

func staticScanner(dir string) (*types.Micro, error) {
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
	m := &types.Micro{
		Name:   name,
		Src:    dir,
		Engine: types.Static,
	}
	return m, nil
}
