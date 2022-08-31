package fs

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func UnzipTemplates(rootZip []byte, dest string, rootDir string) error {
	r, err := zip.NewReader(bytes.NewReader(rootZip), int64(len(rootZip)))
	if err != nil {
		return err
	}

	for _, f := range r.File {
		if !strings.Contains(strings.TrimPrefix(filepath.ToSlash(f.Name), "/"), rootDir) {
			continue
		}

		fpath := strings.ReplaceAll(f.Name, rootDir, dest)

		// make folder if it is a folder
		if f.FileInfo().IsDir() {
			err = os.MkdirAll(fpath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		// make and copy file
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		copyDest, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		srcFile, err := f.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(copyDest, srcFile)
		if err != nil {
			return err
		}

		// close files without defer to close before next iteration of loop
		copyDest.Close()
		srcFile.Close()
	}
	return nil
}

// FileExists returns a bool indicating if a certain file exists in a dir
func FileExists(dir, filename string) (bool, error) {
	info, err := os.Stat(filepath.Join(dir, filename))
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("failed to check if filename %s exists in %s dir: %w", filename, dir, err)
	}

	return !info.IsDir(), nil
}

func IsEmpty(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return true, nil
	}

	f, err := os.Open(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return true, err
		}
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if errors.Is(err, io.EOF) {
		return true, nil
	}

	return false, nil
}

// CheckIfAnyFileExists returns true if any of the filenames exist in a dir
func CheckIfAnyFileExists(dir string, filenames ...string) (bool, error) {
	for _, filename := range filenames {
		exists, err := FileExists(dir, filename)
		if err != nil {
			return false, err
		}
		if exists {
			return true, err
		}
	}
	return false, nil
}
