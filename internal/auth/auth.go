package auth

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

const (
	detaAccessTokenEnv = "DETA_ACCESS_TOKEN"
	detaSignVersion    = "v0"
)

var (
	// ErrNoAccessTokenFound no access token found
	ErrNoAccessTokenFound = errors.New("no access token was found or was empty")
	// ErrInvalidAccessToken invalid access token
	ErrInvalidAccessToken = errors.New("invalid access token")
)

// GetAccessToken gets access token from env var
func GetAccessToken() (string, error) {
	accessToken := os.Getenv(detaAccessTokenEnv)

	if accessToken == "" {
		return "", ErrNoAccessTokenFound
	}

	return accessToken, nil
}

// CalcSignatureInput input to CalcSignature function
type CalcSignatureInput struct {
	AccessToken string
	HTTPMethod  string
	URI         string
	Timestamp   string
	ContentType string
	RawBody     []byte
}

// CalcSignature calculates the signature for signing the requests
func CalcSignature(i *CalcSignatureInput) (string, error) {

	tokenParts := strings.Split(i.AccessToken, "_")
	if len(tokenParts) != 2 {
		return "", ErrInvalidAccessToken
	}
	accessKeyID := tokenParts[0]
	accessKeySecret := tokenParts[1]

	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n",
		i.HTTPMethod,
		i.URI,
		i.Timestamp,
		i.ContentType,
		i.RawBody,
	)

	mac := hmac.New(sha256.New, []byte(accessKeySecret))
	_, err := mac.Write([]byte(stringToSign))
	if err != nil {
		return "", fmt.Errorf("failed to calculate hmac: %w", err)
	}
	signature := mac.Sum(nil)
	hexSign := hex.EncodeToString(signature)

	return fmt.Sprintf("%s=%s:%s", detaSignVersion, accessKeyID, hexSign), nil
}
