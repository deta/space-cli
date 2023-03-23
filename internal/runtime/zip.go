package runtime

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
)

func CompileIgnoreBytes(bytes []byte) (*ignore.GitIgnore, error) {
	s := strings.Split(string(bytes), "\n")
	return ignore.CompileIgnoreLines(s...), nil
}

func ZipDir(sourceDir string) ([]byte, error) {
	absDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for dir %s to zip, %w", sourceDir, err)
	}

	// check if dir exists
	if stat, err := os.Stat(absDir); err != nil && stat.IsDir() {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source dir %s not found, %w", absDir, err)
		}
	}

	var spaceignore *ignore.GitIgnore
	spaceignorePath := filepath.Join(absDir, ".spaceignore")
	if _, err := os.Stat(spaceignorePath); err == nil {
		bytes, err := os.ReadFile(spaceignorePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read .spaceignore: %w", err)
		}

		spaceignore, err = CompileIgnoreBytes(bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to compile .spaceignore: %w", err)
		}
	} else {
		spaceignore, err = CompileIgnoreBytes(defaultSpaceignore)
		if err != nil {
			return nil, fmt.Errorf("failed to compile default .spaceignore: %w", err)
		}
	}

	files := make(map[string][]byte)

	// go through the dir and read all the files
	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip if shouldSkip according to skipPaths which are derived from .spaceignore
		shouldSkip := spaceignore.MatchesPath(path)

		if info.IsDir() {
			if shouldSkip {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkip {
			return nil
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
		contents, e := io.ReadAll(f)
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
