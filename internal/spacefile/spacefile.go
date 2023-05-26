package spacefile

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	_ "embed"

	"github.com/deta/space/pkg/components/emoji"
	"github.com/deta/space/pkg/components/styles"
	"github.com/deta/space/pkg/util/fs"
	"github.com/deta/space/shared"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"gopkg.in/yaml.v3"
)

const (
	// SpacefileName spacefile file name
	SpacefileName = "Spacefile"
)

//go:embed schemas/spacefile.schema.json
var spacefileSchemaString string
var spacefileSchema *jsonschema.Schema = jsonschema.MustCompileString("", spacefileSchemaString)

var (
	ErrSpacefileNotFound = errors.New("Spacefile not found")
	ErrInvalidSpacefile  = errors.New("unable to parse Spacefile")
	ErrDuplicateMicros   = errors.New("micro names have to be unique")
	ErrMultiplePrimary   = errors.New("multiple primary micros present")
	ErrNoPrimaryMicro    = errors.New("no primary micro present")
)

// Spacefile xx
type Spacefile struct {
	V       int             `yaml:"v"`
	Icon    string          `yaml:"icon,omitempty"`
	AppName string          `yaml:"app_name,omitempty"`
	Micros  []*shared.Micro `yaml:"micros,omitempty"`
}

type SpacefileValidationError struct {
	wrapped *jsonschema.ValidationError
	raw     any
}

func extractMicroName(v any, index int) (string, bool) {
	spacefile, ok := v.(map[string]interface{})
	if !ok {
		return "", false
	}

	micros, ok := spacefile["micros"].([]interface{})
	if !ok {
		return "", false
	}

	micro, ok := micros[index].(map[string]interface{})
	if !ok {
		return "", false
	}

	name, ok := micro["name"].(string)
	if !ok {
		return "", false
	}

	return name, true
}

var microReg = regexp.MustCompile(`\/micros\/(\d+)`)

func (ve SpacefileValidationError) Error() string {
	errorMsg := func(leaf *jsonschema.ValidationError) string {
		matches := microReg.FindStringSubmatch(leaf.InstanceLocation)
		if len(matches) == 0 {
			return fmt.Sprintf("L %s: %s", leaf.InstanceLocation, leaf.Message)
		}

		i := matches[1]
		idx, _ := strconv.Atoi(i)
		name, ok := extractMicroName(ve.raw, idx)
		if !ok {
			return fmt.Sprintf("L %s: %s", leaf.InstanceLocation, leaf.Message)
		}

		return fmt.Sprintf("L %s: %s", strings.Replace(leaf.InstanceLocation, i, name, 1), leaf.Message)
	}

	queue := []*jsonschema.ValidationError{ve.wrapped}
	rootErrors := []string{}
	for len(queue) > 0 {
		leaf := queue[0]
		queue = queue[1:]

		if len(leaf.Causes) == 0 {
			rootErrors = append(rootErrors, errorMsg(leaf))
		} else {
			queue = append(queue, leaf.Causes...)
		}
	}

	return fmt.Sprintf("validation failed:\n%s", strings.Join(rootErrors, "\n"))
}

func LoadSpacefile(projectDir string) (*Spacefile, error) {
	spacefilePath := filepath.Join(projectDir, SpacefileName)

	if _, err := os.Stat(spacefilePath); os.IsNotExist(err) {
		return nil, ErrSpacefileNotFound
	} else if err != nil {
		return nil, err
	}

	// read raw contents from spacefile file
	content, err := ioutil.ReadFile(filepath.Join(spacefilePath))
	if err != nil {
		return nil, fmt.Errorf("failed to read contents of spacefile file: %w", err)
	}

	var v any
	if err := yaml.Unmarshal(content, &v); err != nil {
		return nil, ErrInvalidSpacefile
	}

	// validate against schema
	if err := spacefileSchema.Validate(v); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			return nil, SpacefileValidationError{
				wrapped: ve,
				raw:     v,
			}
		}
	}

	var spacefile Spacefile
	if err := yaml.Unmarshal(content, &spacefile); err != nil {
		return nil, ErrInvalidSpacefile
	}

	foundPrimaryMicro := false
	micros := make(map[string]struct{})
	for i, micro := range spacefile.Micros {
		if _, ok := micros[micro.Name]; ok {
			return nil, ErrDuplicateMicros
		}
		micros[micro.Name] = struct{}{}

		if micro.Primary {
			if foundPrimaryMicro {
				return nil, ErrMultiplePrimary
			}

			foundPrimaryMicro = true
			spacefile.Micros[i].Path = "/"
			continue
		}

		if _, err := os.Stat(filepath.Join(projectDir, micro.Src)); os.IsNotExist(err) {
			return nil, fmt.Errorf("micro %s src %s not found", micro.Name, micro.Src)
		}

		if micro.Path != "" {
			if !strings.HasPrefix(micro.Path, "/") {
				micro.Path = fmt.Sprintf("/%s", micro.Path)
			}
			micro.Path = strings.TrimSuffix(micro.Path, "/")

			spacefile.Micros[i].Path = micro.Path
			continue
		}

		spacefile.Micros[i].Path = fmt.Sprintf("/%s", micro.Name)
	}

	if !foundPrimaryMicro {
		if len(spacefile.Micros) == 1 {
			spacefile.Micros[0].Primary = true
			spacefile.Micros[0].Path = "/"
		} else {
			return nil, ErrNoPrimaryMicro
		}
	}

	return &spacefile, nil
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
	var rawSpacefile bytes.Buffer
	yamlEncoder := yaml.NewEncoder(&rawSpacefile)
	yamlEncoder.SetIndent(2)
	err := yamlEncoder.Encode(&s)
	if err != nil {
		return fmt.Errorf("failed to marshall spacefile object: %w", err)
	}

	c := []byte(spacefileDocsUrl)
	c = append(c, rawSpacefile.Bytes()...)

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
	if s.Icon == "" {
		return nil, ErrInvalidIconPath
	}
	iconMeta, err := getIconMeta(s.Icon)
	if err != nil {
		return nil, err
	}

	raw, err := os.ReadFile(filepath.Join(s.Icon))
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

func ParseSpacefileUnmarshallTypeError(err *yaml.TypeError) string {
	errMsg := styles.Errorf("%sError: failed to parse your Spacefile, please make sure you use the correct syntax:", emoji.ErrorExclamation)
	for _, err := range err.Errors {
		fieldNotValidMatches := regexp.MustCompile(`(?m)(line \d:) field\s(\w+)\snot found in type.*`).FindStringSubmatch(err)
		if len(fieldNotValidMatches) > 0 {
			errMsg += fmt.Sprintf("\n  L %v \"%v\" is not a valid field\n", fieldNotValidMatches[1], fieldNotValidMatches[2])
		} else {
			errMsg += styles.Boldf("\n  L %v\n", err)
		}
	}
	return errMsg
}
