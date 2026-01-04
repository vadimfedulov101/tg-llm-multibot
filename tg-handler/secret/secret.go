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

	keysStr := string(content)
	if strings.TrimSpace(keysStr) == "" {
		log.Fatal(ErrEmptyKeysStr)
	}

	// Split by newline and clean up
	rawLines := strings.Split(keysStr, "\n")
	var keys []string

	for _, line := range rawLines {
		// TrimSpace removes \t, \n, \r, and spaces
		cleaned := strings.TrimSpace(line)
		if cleaned != "" {
			keys = append(keys, cleaned)
		}
	}

	if len(keys) < 1 {
		log.Fatal(ErrZeroKeys)
	}

	return keys
}
