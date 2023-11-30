package scanner

import (
	"fmt"

	"github.com/deta/space/pkg/util/fs"
	"github.com/deta/space/shared"
)

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
		Engine: shared.Python311,
	}

	return m, nil
}

func nodeScanner(dir string) (*shared.Micro, error) {
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

	m := &shared.Micro{
		Name:   name,
		Src:    dir,
		Engine: shared.Node20x,
	}

	framework, err := detectFramework(dir)
	if err != nil {
		return nil, err
	}
	m.Engine = framework

	return m, nil
}

func goScanner(dir string) (*shared.Micro, error) {
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
		Serve:  "./",
		Engine: shared.Static,
	}
	return m, nil
}
