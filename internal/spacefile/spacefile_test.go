package spacefile

import (
	"errors"
	"testing"
)

type TestCase struct {
	projectDir    string
	expectedError error
}

func TestValidation(t *testing.T) {
	cases := []TestCase{
		{
			projectDir:    "testdata/spacefile/duplicated_micros",
			expectedError: ErrDuplicateMicros,
		},
		{
			projectDir:    "testdata/spacefile/invalid_micro_path",
			expectedError: ErrSpacefileNotFound,
		},
		{
			projectDir:    "testdata/spacefile/multiple_primary",
			expectedError: ErrMultiplePrimary,
		},
		{
			projectDir:    "testdata/spacefile/no_primary",
			expectedError: ErrNoPrimaryMicro,
		},
		{
			projectDir:    "testdata/spacefile/single_micro",
			expectedError: nil,
		},
		{
			projectDir:    "testdata/spacefile/multiple_micros",
			expectedError: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.projectDir, func(t *testing.T) {
			_, err := LoadSpacefile(c.projectDir)

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
	spacefile := "./testdata/spacefile/implicit_primary"
	space, err := LoadSpacefile(spacefile)
	if err != nil {
		t.Fatalf("failed to parse spacefile: %v", err)
	}

	if !space.Micros[0].Primary {
		t.Fatalf("expected primary to be true but got false")
	}
}
