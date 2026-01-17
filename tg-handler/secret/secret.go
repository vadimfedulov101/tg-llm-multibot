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
		"[secret] failed to get '" + envVar + "' environment variable",
	)
	ErrReadFileFailed = errors.New(
		"[secret] failed to read '%s' file",
	)
	ErrEmptyKeysStr = errors.New(
		"[secret] got empty keys string from '%s' file",
	)
	ErrZeroKeys = errors.New(
		"[secret] got zero keys from '%s' file",
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
	if strings.TrimSpace(keysStr) == "" {
		log.Fatal(ErrEmptyKeysStr)
	}

	// Split into raw lines by "\n"
	rawLines := strings.Split(keysStr, "\n")

	// Accumulate keys as cleaned lines
	var keys []string
	for _, rawLine := range rawLines {
		cleanedLine := strings.TrimSpace(rawLine)
		if cleanedLine != "" {
			keys = append(keys, cleanedLine)
		}
	}

	// Check if got non-zero keys
	if len(keys) < 1 {
		log.Fatal(ErrZeroKeys)
	}

	return keys
}
