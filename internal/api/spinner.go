package api

import (
	"fmt"
)

const arrows = "|\\-/"

type ReadSpinner struct {
	name   string
	length int64
	read   int64
	index  int
}

func NewReadSpinner(name string, length int64) *ReadSpinner {
	return &ReadSpinner{name, length, 0, 0}
}

func (s *ReadSpinner) next() string {
	s.index = (s.index + 1) % len(arrows)
	return arrows[s.index : s.index+1]
}

func (s *ReadSpinner) Status(read int) {
	s.read += int64(read)
	var percentage int64
	if s.length != 0 {
		percentage = s.read * 100 / s.length
	}
	fmt.Printf(
		"[%s] %s%02d%% %d / %d\r", s.next(), s.name,
		percentage, s.read, s.length,
	)
}
