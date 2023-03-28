package scanner

import (
	"testing"

	"github.com/deta/pc-cli/types"
	"golang.org/x/exp/slices"
	"gotest.tools/v3/assert"
)

type ScanTestInfo struct {
	Name           string
	Path           string
	ExpectedEngine string
}

var (
	microsTestInfo = []ScanTestInfo{
		{Name: "python", Path: "testdata/micros/python", ExpectedEngine: types.Python39},
		{Name: "go", Path: "testdata/micros/go", ExpectedEngine: types.Custom},
		{Name: "next", Path: "testdata/micros/next", ExpectedEngine: types.Next},
		{Name: "node", Path: "testdata/micros/node", ExpectedEngine: types.Node16x},
		{Name: "nuxt", Path: "testdata/micros/nuxt", ExpectedEngine: types.Nuxt},
		{Name: "react", Path: "testdata/micros/react", ExpectedEngine: types.React},
		{Name: "static", Path: "testdata/micros/static", ExpectedEngine: types.Static},
		{Name: "svelte", Path: "testdata/micros/svelte", ExpectedEngine: types.Svelte},
		{Name: "svelte-kit", Path: "testdata/micros/svelte-kit", ExpectedEngine: types.SvelteKit},
		{Name: "vue", Path: "testdata/micros/vue", ExpectedEngine: types.Vue},
	}
)

func TestScanSingleMicroProjects(t *testing.T) {
	for _, project := range microsTestInfo {
		micros, err := Scan(project.Path)
		if err != nil {
			t.Fatalf("failed to scan project %s at %s while testing, %v", project.Name, project.Path, err)
		}
		assert.Equal(t, len(micros), 1, "detected multiple micros in a single micro project")
		micro := micros[0]
		assert.Equal(t, micro.Engine, project.ExpectedEngine, "detected engine as %s but expected %s", micro.Engine, project.ExpectedEngine)
	}
}

func TestScanMultiMicroProject(t *testing.T) {

	expectedMicros := []string{"python", "go", "next", "node", "nuxt", "react", "static", "svelte", "svelte-kit", "vue"}
	expectedMicrosToEngines := map[string]string{
		"python":     types.Python39,
		"go":         types.Custom,
		"next":       types.Next,
		"node":       types.Node16x,
		"nuxt":       types.Nuxt,
		"react":      types.React,
		"static":     types.Static,
		"svelte":     types.Svelte,
		"svelte-kit": types.SvelteKit,
		"vue":        types.Vue,
	}

	sourceDir := "testdata/micros"

	micros, err := Scan(sourceDir)
	if err != nil {
		t.Fatalf("failed to scan project at %s while testing multi micros auto-detection, %v", sourceDir, err)
	}

	assert.Equal(t, len(micros), len(expectedMicros), "detected %d micros, but expected %d", len(micros), len(expectedMicros))

	for _, micro := range micros {
		if !slices.Contains(expectedMicros, micro.Name) {
			t.Fatalf("micro %s at %s is detected, but should not be detected as part of a multi-micro project", micro.Name, micro.Src)
		}
		assert.Equal(t, micro.Engine, expectedMicrosToEngines[micro.Name], "detected engine for micro %s as %s, but expected %s", micro.Name, micro.Engine, expectedMicrosToEngines[micro.Name])
	}
}

func TestEmptyProject(t *testing.T) {
	sourceDir := "testdata/empty"

	micros, err := Scan(sourceDir)
	if err != nil {
		t.Fatalf("failed to scan project at %s while testing empty project auto-detection, %v", sourceDir, err)
	}

	assert.Equal(t, 0, len(micros), "detected micros in empty project")
}
