package runtime

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func shouldSkip(path string) bool {
	skipPaths := []string{"node_modules"}

	for _, skipPath := range skipPaths {
		if strings.Contains(path, skipPath) {
			return true
		}
	}
	return false
}

func ZipDir(sourceDir string) ([]byte, error) {

	// TODO: check if dir exists

	absDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for dir %s to zip, %w", sourceDir, err)
	}

	files := make(map[string][]byte)

	// go through the dir and read all the files
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			// skip directory if code package (E.g. node_modules)
			shouldSkip := shouldSkip(path)
			if shouldSkip {
				return filepath.SkipDir
			}
			return nil
		}

		if err != nil {
			return err
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}

		// relative path of file from absolute locations of dir and path
		relPath, err := filepath.Rel(absDir, absPath)
		if err != nil {
			return err
		}

		// ensures to use forward slashes
		relPath = filepath.ToSlash(relPath)

		f, e := os.Open(path)
		if e != nil {
			return e
		}
		defer f.Close()
		contents, e := ioutil.ReadAll(f)
		if e != nil {
			return e
		}

		files[relPath] = contents
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("cannot scan contents of dir %s to zip, %w", sourceDir, err)
	}

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			return nil, fmt.Errorf("cannot compress file %s of dir %s, %w", name, sourceDir, err)
		}
		_, err = f.Write(content)
		if err != nil {
			return nil, fmt.Errorf("cannot compress file %s of dir %s, %w", name, sourceDir, err)
		}
	}

	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("cannot close zip writer for dir %s, %w", sourceDir, err)
	}

	return buf.Bytes(), nil
}
