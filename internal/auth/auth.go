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
	spaceAccessTokenEnv         = "SPACE_ACCESS_TOKEN"
	spaceTokensFile             = "space_tokens"
	spaceSignVersion            = "v0"
	spaceDir                    = ".detaspace"
	oldSpaceDir                 = ".deta"
	dirModePermReadWriteExecute = 0760
	fileModePermReadWrite       = 0660
	spaceProjectKeysPath        = ".detaspace/space_project_keys"
)

var (
	spaceAuthTokenPath    = filepath.Join(spaceDir, spaceTokensFile)
	oldSpaceAuthTokenPath = filepath.Join(oldSpaceDir, spaceTokensFile)

	// ErrNoProjectKeyFound no access token found
	ErrNoProjectKeyFound = errors.New("no project key was found or was empty")
	// ErrNoAccessTokenFound no access token found
	ErrNoAccessTokenFound = errors.New("no access token was found or was empty")
	// ErrInvalidAccessToken invalid access token
	ErrInvalidAccessToken = errors.New("invalid access token")
	// ErrBadAccessTokenFile bad access token file
	ErrBadAccessTokenFile = errors.New("bad access token file")
)

type Token struct {
	AccessToken string `json:"access_token"`
}

func getAccessTokenFromFile(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("os.Open: %w", err)
	}
	defer f.Close()
	var t Token
	if err := json.NewDecoder(f).Decode(&t); err != nil {
		return "", fmt.Errorf("%w: %s", ErrBadAccessTokenFile, filepath)
	}
	if t.AccessToken == "" {
		return t.AccessToken, ErrNoAccessTokenFound
	}
	return t.AccessToken, nil
}

// GetAccessToken retrieves the tokens from storage or env var
func GetAccessToken() (string, error) {
	// preference to env var first
	spaceAccessToken := os.Getenv(spaceAccessTokenEnv)
	if spaceAccessToken != "" {
		return spaceAccessToken, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	tokensFilePath := filepath.Join(home, spaceAuthTokenPath)
	accessToken, err := getAccessTokenFromFile(tokensFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return accessToken, fmt.Errorf("failed to get access token from file: %w", err)
		}
		// fallback to old space auth token path
		tokensFilePath = filepath.Join(home, oldSpaceAuthTokenPath)
		accessToken, err = getAccessTokenFromFile(tokensFilePath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return accessToken, fmt.Errorf("failed to get access token from file: %w", err)
			}
			return "", ErrNoAccessTokenFound
		}
		// store access token in new token directory if old directory
		if err := StoreAccessToken(accessToken); err != nil {
			return "", fmt.Errorf("failed to store access token from old token path to new path: %w", err)
		}
	}
	return accessToken, nil
}

func storeAccessToken(t *Token, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirModePermReadWriteExecute); err != nil {
		return fmt.Errorf("failed to create dir %s: %w", dir, err)
	}
	marshalled, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("failed to marshall token: %w", err)
	}
	if err := os.WriteFile(path, marshalled, fileModePermReadWrite); err != nil {
		return fmt.Errorf("failed to write token to file %s: %w", path, err)
	}
	return nil
}

// StoreAccessToken in the access token directory
func StoreAccessToken(accessToken string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	tokensFilePath := filepath.Join(home, spaceAuthTokenPath)
	t := &Token{AccessToken: accessToken}
	if err := storeAccessToken(t, tokensFilePath); err != nil {
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

type Keys map[string]interface{}

// GetProjectKey retrieves a project key storage or env var
func GetProjectKey(projectId string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", nil
	}

	keysFilePath := filepath.Join(home, spaceProjectKeysPath)
	f, err := os.Open(keysFilePath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	defer f.Close()

	var keys Keys
	contents, _ := ioutil.ReadAll(f)
	json.Unmarshal(contents, &keys)

	var key = keys[projectId]
	if key != nil {
		return key.(string), nil
	}

	return "", ErrNoProjectKeyFound
}

func StoreProjectKey(projectId string, projectKey string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	spaceDirPath := filepath.Join(home, spaceDir)
	err = os.MkdirAll(spaceDirPath, 0760)
	if err != nil {
		return err
	}

	keys := make(map[string]interface{})
	keys[projectId] = projectKey

	marshalled, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	keysFilePath := filepath.Join(home, spaceProjectKeysPath)
	err = ioutil.WriteFile(keysFilePath, marshalled, 0660)
	if err != nil {
		return err
	}
	return nil
}
