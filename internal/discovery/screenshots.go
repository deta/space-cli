package discovery

import (
	"errors"
	"io"
	"mime"
	"net/url"
	"os"
	"path/filepath"
)

var (
	// ErrInvalidScreenshotPath cannot find screenshot path
	ErrInvalidScreenshotPath = errors.New("invalid screenshot path")
)

// Screenshot xx
type Screenshot struct {
	Raw         []byte `json:"screenshot"`
	ContentType string `json:"content_type"`
}

func ParseScreenshot(paths []string) ([]Screenshot, error) {
	screenshots := make([]Screenshot, 0)

	for _, path := range paths {
		screenshot := Screenshot{}
		if isValidVideoURL(path) {
			screenshot.Raw = []byte(path)
			screenshot.ContentType = "text/plain"
		} else {
			absPath, err := filepath.Abs(path)
			if err != nil {
				return screenshots, ErrInvalidScreenshotPath
			}

			isdir, err := isDir(&absPath)
			if err != nil {
				return screenshots, err
			}

			if isdir {
				// get file names in the directory
				inFiles, err := getFilesInDirectory(absPath)
				if err != nil {
					return screenshots, err
				}

				// recursive call
				res, err := ParseScreenshot(inFiles)
				if err != nil {
					return screenshots, err
				}
				return res, nil
			}

			file, err := os.Open(absPath)
			if err != nil {
				return screenshots, ErrInvalidScreenshotPath
			}
			defer file.Close()
			content, err := io.ReadAll(file)
			if err != nil {
				return screenshots, err
			}
			screenshot.Raw = content

			ext := filepath.Ext(absPath)
			screenshot.ContentType = mime.TypeByExtension(ext)
		}
		screenshots = append(screenshots, screenshot)
	}

	return screenshots, nil
}

// getFilesInDirectory xx
func getFilesInDirectory(directoryPath string) ([]string, error) {
	var filePaths []string

	files, err := os.ReadDir(directoryPath)
	if err != nil {
		return nil, ErrInvalidScreenshotPath
	}

	for _, file := range files {
		filePath := filepath.Join(directoryPath, file.Name())
		if file.IsDir() {
			continue
		}
		filePaths = append(filePaths, filePath)
	}

	return filePaths, nil
}

// isDir xx
func isDir(path *string) (bool, error) {
	fileInfo, err := os.Stat(*path)
	if err != nil {
		return false, ErrInvalidScreenshotPath
	}

	if fileInfo.IsDir() {
		return true, nil
	}
	return false, nil
}

// isValidVideoURL xx
func isValidVideoURL(sURL string) bool {
	_, err := url.ParseRequestURI(sURL)
	if err != nil {
		return false
	}
	return true
}
