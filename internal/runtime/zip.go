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

func ZipDir(sourceDir string) ([]byte, int, error) {
	absDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to resolve absolute path for dir %s to zip, %w", sourceDir, err)
	}

	// check if dir exists
	if stat, err := os.Stat(absDir); err != nil && stat.IsDir() {
		if os.IsNotExist(err) {
			return nil, 0, fmt.Errorf("source dir %s not found, %w", absDir, err)
		}
	}

	lines := strings.Split(string(defaultSpaceignore), "\n")
	spaceIgnorePath := filepath.Join(sourceDir, spaceignoreFile)
	if _, err := os.Stat(spaceIgnorePath); err == nil {
		bytes, err := os.ReadFile(spaceIgnorePath)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read .spaceignore: %w", err)
		}
		lines = append(lines, strings.Split(string(bytes), "\n")...)
	}

	spaceignore := ignore.CompileIgnoreLines(lines...)

	files := make(map[string][]byte)
	// go through the dir and read all the files
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip if shouldSkip according to skipPaths which are derived from .spaceignore
		shouldSkip := spaceignore.MatchesPath(path)
		if shouldSkip && info.IsDir() {
			return filepath.SkipDir
		}

		if shouldSkip {
			return nil
		}

		if info.IsDir() {
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
		return nil, 0, fmt.Errorf("cannot scan contents of dir %s to zip, %w", sourceDir, err)
	}

	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	filenames := make([]string, 0, len(files))
	for name, content := range files {
		filenames = append(filenames, name)
		f, err := w.Create(name)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot compress file %s of dir %s, %w", name, sourceDir, err)
		}
		_, err = f.Write(content)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot compress file %s of dir %s, %w", name, sourceDir, err)
		}
	}

	err = w.Close()
	if err != nil {
		return nil, 0, fmt.Errorf("cannot close zip writer for dir %s, %w", sourceDir, err)
	}

	return buf.Bytes(), len(filenames), nil
}
