package secret

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"tg-handler/logging"
)

const (
	envVar = "API_KEYS_FILE"
)

// Secret errors
var (
	errGetEnvFailed   = errors.New("failed to get env variable")
	errReadFileFailed = errors.New("failed to read file")
	errEmptyKeysStr   = errors.New("got empty keys string")
	errZeroKeys       = errors.New("got zero keys")
)

// Loads API keys from environment variable or panics
func MustLoadAPIKeys(logger *logging.Logger) []string {
	const errMsg = "failed to load API keys"

	// Get secret file path
	logger = logger.With(logging.EnvVar(envVar))
	path, ok := os.LookupEnv(envVar)
	if !ok {
		logger.Panic(errMsg, logging.Err(errGetEnvFailed))
	}
	logger = logger.With(logging.Path(path))

	// Read secret file from path
	content, err := os.ReadFile(path)
	if err != nil {
		logger.Panic(errMsg, logging.Err(
			fmt.Errorf("%w: %v", errReadFileFailed, err)),
		)
	}

	// Get non-empty keys string
	keysStr := string(content)
	if strings.TrimSpace(keysStr) == "" {
		logger.Panic(errMsg, logging.Err(errEmptyKeysStr))
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
		logger.Panic(errMsg, logging.Err(errZeroKeys))
	}

	return keys
}
