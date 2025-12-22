package secret

import (
	"errors"
	"log"
	"os"
	"strings"
)

const (
	envVar = "API_KEYS_FILE"
)

// Secret errors
var (
	ErrGetEnvFailed = errors.New(
		"Failed to get '" + envVar + "' environment variable",
	)
	ErrReadFileFailed = errors.New(
		"Failed to read '%s' file",
	)
	ErrEmptyKeysStr = errors.New(
		"Got empty keys string from '%s' file",
	)
	ErrZeroKeys = errors.New(
		"Got zero keys from '%s' file",
	)
)

// Loads API keys from environment variable or panics
func MustLoadAPIKeys() []string {
	// Get secret file from environment variable
	secretFile, ok := os.LookupEnv(envVar)
	if !ok {
		log.Fatal(ErrGetEnvFailed)
	}

	// Read secret file
	content, err := os.ReadFile(secretFile)
	if err != nil {
		log.Fatalf("%v: %v", ErrReadFileFailed, err)
	}

	// Get non-empty keys string
	keysStr := string(content)
	if keysStr == "" {
		log.Fatal(ErrEmptyKeysStr)
	}

	// Get non-zero keys
	keys := strings.Split(keysStr, "\n")
	if len(keys) < 1 {
		log.Fatal(ErrZeroKeys)
	}

	return keys
}
