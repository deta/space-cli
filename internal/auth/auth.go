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
)

var (
	spaceAuthTokenPath    string
	oldSpaceAuthTokenPath string
	spaceProjectKeysPath  string
	spaceApiKeysPath      string

	// ErrNoProjectKeyFound no access token found
	ErrNoProjectKeyFound = errors.New("no project key was found or was empty")
	// ErrNoAccessTokenFound no access token found
	ErrNoAccessTokenFound = errors.New("no access token was found or was empty")
	// ErrInvalidAccessToken invalid access token
	ErrInvalidAccessToken = errors.New("invalid access token")
	// ErrBadAccessTokenFile bad access token file
	ErrBadAccessTokenFile = errors.New("bad access token file")
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	spaceProjectKeysPath = filepath.Join(home, spaceDir, "space_project_keys")
	spaceApiKeysPath = filepath.Join(home, spaceDir, "space_api_keys")
	spaceAuthTokenPath = filepath.Join(home, spaceDir, spaceTokensFile)
	oldSpaceAuthTokenPath = filepath.Join(home, oldSpaceDir, spaceTokensFile)
}

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

	tokensFilePath := spaceAuthTokenPath
	accessToken, err := getAccessTokenFromFile(tokensFilePath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return accessToken, fmt.Errorf("failed to get access token from file: %w", err)
		}
		// fallback to old space auth token path
		tokensFilePath = oldSpaceAuthTokenPath
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

type Keys map[string]string

// GetKey retrieves a project key storage or env var
func GetKey(keysFilePath string, keyName string) (string, error) {
	f, err := os.Open(keysFilePath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	defer f.Close()

	var keys Keys
	if err := json.NewDecoder(f).Decode(&keys); err != nil {
		return "", err
	}

	if key, ok := keys[keysFilePath]; ok {
		return key, nil
	}

	return "", ErrNoProjectKeyFound
}

func StoreKey(keysFilePath string, keyName string, keyValue string) error {
	if _, err := os.Stat(keysFilePath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(keysFilePath), dirModePermReadWriteExecute); err != nil {
			return err
		}
	}

	keys := make(map[string]interface{})
	keys[keyName] = keyValue

	marshalled, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(keysFilePath, marshalled, 0660)
	if err != nil {
		return err
	}
	return nil
}

func GetProjectKey(projectID string) (string, error) {
	return GetKey(spaceProjectKeysPath, projectID)
}

func GetApiKey(hostname string) (string, error) {
	return GetKey(spaceApiKeysPath, hostname)
}

func StoreProjectKey(name string, value string) error {
	return StoreKey(spaceProjectKeysPath, name, value)
}

func StoreApiKey(name string, value string) error {
	return StoreKey(spaceApiKeysPath, name, value)
}
