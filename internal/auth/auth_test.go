package auth

import (
	"fmt"
	"testing"
)

func TestSignature(t *testing.T) {
	testCases := []struct {
		input    *CalcSignatureInput
		expected string
	}{
		{
			input: &CalcSignatureInput{
				// This is a dummy access token, don't worry
				AccessToken: "xkcfKpsU_zwDNmNSqG9TGEiR8sSm8HVrSWuJ31b4d",
				URI:         "/api/v0/space",
				Timestamp:   "1681294911",
				HTTPMethod:  "GET",
				RawBody:     nil,
			},
			expected: "v0=xkcfKpsU:120071dd2ce1ca9cc08f76efe4e3b01179ef76ab20e6ad4655b79abdf995146b",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("signature %d", i), func(t *testing.T) {
			actual, err := CalcSignature(tc.input)
			if err != nil {
				t.Errorf("expected no error, actual: %s", err)
			}

			if actual != tc.expected {
				t.Errorf("expected: %s, actual: %s", tc.expected, actual)
			}
		})
	}
}
