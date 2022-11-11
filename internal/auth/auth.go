package auth

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

const (
	spaceAccessTokenEnv   = "SPACE_ACCESS_TOKEN"
	spaceSignVersion      = "v0"
	spaceDir              = ".detaspace"
	spaceAuthTokenPath    = ".detaspace/space_tokens"
	oldSpaceAuthTokenPath = ".deta/space_tokens"
)

var (
	// ErrNoAccessTokenFound no access token found
	ErrNoAccessTokenFound = errors.New("no access token was found or was empty")
	// ErrInvalidAccessToken invalid access token
	ErrInvalidAccessToken = errors.New("invalid access token")
)

type Token struct {
	AccessToken string `json:"access_token"`
}

// GetAccessToken retrieves the tokens from storage or env var
func GetAccessToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", nil
	}

	tokensFilePath := filepath.Join(home, spaceAuthTokenPath)
	f, err := os.Open(tokensFilePath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	defer f.Close()

	// ignoring errors here
	// as we fall back to retrieving acces token from env
	// if not found in env then will finally return an error
	var tokens Token
	contents, _ := ioutil.ReadAll(f)
	json.Unmarshal(contents, &tokens)

	// first priority to access token
	if tokens.AccessToken != "" {
		return tokens.AccessToken, nil
	}

	// check old token file path and move to new token file path
	// ignore errors here because this is old token file path
	oldTokensFilePath := filepath.Join(home, oldSpaceAuthTokenPath)
	of, _ := os.Open(oldTokensFilePath)
	defer of.Close()
	var oldTokens Token
	contents, _ = ioutil.ReadAll(of)
	json.Unmarshal(contents, &oldTokens)
	if oldTokens.AccessToken != "" {
		StoreAccessToken(oldTokens.AccessToken)
		return oldTokens.AccessToken, nil
	}

	// not found in file, check the env
	spaceAccessToken := os.Getenv(spaceAccessTokenEnv)

	if spaceAccessToken != "" {
		return spaceAccessToken, nil
	}

	return "", ErrNoAccessTokenFound
}

func StoreAccessToken(accessToken string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	spaceDirPath := filepath.Join(home, spaceDir)
	err = os.MkdirAll(spaceDirPath, 0760)
	if err != nil {
		return err
	}

	var tokens = &Token{AccessToken: accessToken}
	marshalled, err := json.Marshal(tokens)
	if err != nil {
		return err
	}

	tokensFilePath := filepath.Join(home, spaceAuthTokenPath)
	err = ioutil.WriteFile(tokensFilePath, marshalled, 0660)
	if err != nil {
		return err
	}
	return nil
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

	return fmt.Sprintf("%s=%s:%s", spaceSignVersion, accessKeyID, hexSign), nil
}
