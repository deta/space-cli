package spacefile

import (
	"os"
	"testing"
)

type TestCase struct {
	spacefile string
	valid     bool
}

func TestValidation(t *testing.T) {
	cases := []TestCase{
		{
			spacefile: "./testdata/spacefile/missing_field.yaml",
			valid:     false,
		},
		{
			spacefile: "./testdata/spacefile/single_micro.yaml",
			valid:     true,
		},
		{
			spacefile: "./testdata/spacefile/multiple_micros.yaml",
			valid:     true,
		},
	}

	for _, c := range cases {
		t.Run(c.spacefile, func(t *testing.T) {

			content, err := os.ReadFile(c.spacefile)
			if err != nil {
				t.Fatalf("failed to read spacefile: %v", err)
			}

			if err := ValidateSpacefileStructure(content); err != nil && c.valid {
				t.Fatalf("expected valid spacefile but got error: %v", err)
			} else if err == nil && !c.valid {
				t.Fatalf("expected invalid spacefile but got no error")
			}

		})
	}

}
