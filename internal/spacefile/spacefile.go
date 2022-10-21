package spacefile

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/deta/pc-cli/pkg/util/fs"
	"github.com/deta/pc-cli/shared"
	"gopkg.in/yaml.v3"
)

const (
	// SpacefileName spacefile file name
	SpacefileName = "Spacefile"
)

func Open(sourceDir string) (*Spacefile, error) {
	var exists bool
	var err error

	exists, err = fs.FileExists(sourceDir, SpacefileName)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrSpacefileNotFound
	}

	// read raw contents from spacefile file
	c, err := ioutil.ReadFile(filepath.Join(sourceDir, SpacefileName))
	if err != nil {
		return nil, fmt.Errorf("failed to read contents of spacefile file: %w", err)
	}

	// parse raw spacefile file content
	s := Spacefile{}
	dec := yaml.NewDecoder(bytes.NewReader(c))
	dec.KnownFields(true)

	err = dec.Decode(&s)
	if err != nil {
		return nil, fmt.Errorf("failed to do parse spacefile file, please check for correct syntax: %w", err)
	}

	return &s, nil
}

// OpenRaw returns the raw spacefile file content from sourceDir if it exists
func OpenRaw(sourceDir string) ([]byte, error) {
	var exists bool
	var err error

	exists, err = fs.FileExists(sourceDir, SpacefileName)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrSpacefileNotFound
	}

	// read raw contents from spacefile file
	c, err := ioutil.ReadFile(filepath.Join(sourceDir, SpacefileName))
	if err != nil {
		return nil, fmt.Errorf("failed to read contents of spacefile file: %w", err)
	}

	return c, nil
}

func (s *Spacefile) Save(sourceDir string) error {

	spacefileDocsUrl := "# Spacefile Docs: https://go.deta.dev/docs/spacefile/v0\n"
	// marshall spacefile object
	rawSpacefile, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("failed to marshall spacefile object: %w", err)
	}

	c := []byte(spacefileDocsUrl)
	c = append(c, rawSpacefile...)

	// write spacefile object to file
	err = ioutil.WriteFile(filepath.Join(sourceDir, SpacefileName), c, 0644)
	if err != nil {
		return fmt.Errorf("failed to write spacefile object: %w", err)
	}

	return nil
}

func (s *Spacefile) AddMicros(newMicros []*shared.Micro) error {
	for _, micro := range newMicros {
		if err := s.AddMicro(micro); err != nil {
			return fmt.Errorf("failed to add micro %s to spacefile, %w", micro.Name, err)
		}
	}
	return nil
}

func (s *Spacefile) GetIcon() (*Icon, error) {

	iconMeta, err := getIconMeta(s.Icon)
	if err != nil {
		return nil, err
	}

	raw, err := ioutil.ReadFile(filepath.Join(s.Icon))
	if err != nil {
		return nil, fmt.Errorf("cannot read image, %w", err)
	}

	return &Icon{Raw: raw, IconMeta: iconMeta}, nil
}

func (s *Spacefile) AddMicro(newMicro *shared.Micro) error {
	// mark new micro as primary if it is the only one
	if len(s.Micros) == 0 {
		newMicro.Primary = true
	}

	for _, micro := range s.Micros {
		if micro.Name == newMicro.Name {
			return fmt.Errorf("a micro with the same name already exists in \"Spacefile\"")
		}
		if micro.Src == newMicro.Src {
			return fmt.Errorf("another micro already exists at the same location %s in the spacefile", newMicro.Src)
		}
	}
	s.Micros = append(s.Micros, newMicro)

	return nil
}

func CreateSpacefileWithMicros(sourceDir string, micros []*shared.Micro) (*Spacefile, error) {
	// mark one micro as primary
	if len(micros) > 0 {
		micros[0].Primary = true
	}

	s := new(Spacefile)
	s.Micros = make([]*shared.Micro, len(micros))
	copy(s.Micros, micros)

	err := s.Save(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create spacefile with micros in %s, %w", sourceDir, err)
	}

	return s, nil
}

func CreateBlankSpacefile(sourceDir string) (*Spacefile, error) {
	s := new(Spacefile)

	err := s.Save(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create a blank spacefile in %s, %w", sourceDir, err)
	}

	return s, nil
}

func (s *Spacefile) HasMicro(otherMicro *shared.Micro) bool {
	for _, micro := range s.Micros {
		if micro.Name == otherMicro.Name && micro.Src == otherMicro.Src {
			return true
		}
	}
	return false
}

func IsSpacefilePresent(sourceDir string) (bool, error) {
	var exists bool
	var err error

	exists, err = fs.FileExists(sourceDir, SpacefileName)
	if err != nil {
		return false, err
	}

	if exists {
		return true, nil
	}

	return false, nil
}
