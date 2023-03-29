package spacefile

import (
	"errors"
	"testing"
)

type TestCase struct {
	spacefile     string
	expectedError error
}

func TestValidation(t *testing.T) {
	cases := []TestCase{
		{
			spacefile:     "testdata/spacefile/duplicate_micros.yaml",
			expectedError: ErrDuplicateMicros,
		},
		{
			spacefile:     "testdata/spacefile/invalid_micro_path.yaml",
			expectedError: ErrSpacefileNotFound,
		},
		{
			spacefile:     "testdata/spacefile/multiple_primary.yaml",
			expectedError: ErrMultiplePrimary,
		},
		{
			spacefile:     "testdata/spacefile/no_primary.yaml",
			expectedError: ErrNoPrimaryMicro,
		},
		{
			spacefile:     "testdata/spacefile/single_micro.yaml",
			expectedError: nil,
		},
		{
			spacefile:     "testdata/spacefile/multiple_micros.yaml",
			expectedError: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.spacefile, func(t *testing.T) {
			_, err := ParseSpacefile(c.spacefile)

			if err != nil && c.expectedError == nil {
				t.Fatalf("expected no error but got: %v", err)
			}

			if err == nil && c.expectedError != nil {
				t.Fatalf("expected error but got none")
			}

			if !errors.Is(err, c.expectedError) {
				t.Fatalf("expected error to be %v but got %v", c.expectedError, err)
			}
		})
	}
}

func TestImplicitPrimary(t *testing.T) {
	spacefile := "./testdata/spacefile/implicit_primary.yaml"
	space, err := ParseSpacefile(spacefile)
	if err != nil {
		t.Fatalf("failed to parse spacefile: %v", err)
	}

	if !space.Micros[0].Primary {
		t.Fatalf("expected primary to be true but got false")
	}
}
