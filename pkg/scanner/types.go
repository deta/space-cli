package scanner

import "github.com/deta/pc-cli/types"

type engineScanner func(dir string) (*types.Micro, error)

type Match struct {
	Path         string
	MatchContent string
}

type Detectors struct {
	Matches []Match
	Strict  bool // strict requires all the matches to pass
}

type NodeFramework struct {
	Name      string
	Detectors Detectors
}
