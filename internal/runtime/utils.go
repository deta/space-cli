package runtime

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
)

func readLines(data []byte) ([]string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, scanner.Err()
}

// reads the contents of a file, returns contents
func (m *Manager) readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ioutil.ReadAll(f)
}
