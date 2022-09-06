package runtime

import "encoding/json"

// ProjectMeta xx
type ProjectMeta struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

// unmarshals data into a ProjectMeta
func projectMetaFromBytes(data []byte) (*ProjectMeta, error) {
	var p ProjectMeta
	err := json.Unmarshal(data, &p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}
