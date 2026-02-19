package translator

import (
	"errors"
	"fmt"
	"time"

	"github.com/bregydoc/gtranslate"
)

var (
	errMsgTranslationFailed = errors.New(
		"failed to translate from English",
	)
)

const (
	maxRetries = 10
	retryDelay = 5 * time.Second
)

// Translates original to language based on its tag
func Translate(original string) (string, error) {
	var err error
	var translated string

	for i := range maxRetries {
		translated, err = gtranslate.TranslateWithParams(
			original,
			gtranslate.TranslationParams{
				From: "en",
				To:   "eo",
			},
		)

		// If success, return immediately
		if err == nil {
			return translated, nil
		}

		// If failed, wait before retrying (unless last attempt)
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	// Return the last error if all retries failed
	return "", fmt.Errorf("%w: %v", errMsgTranslationFailed, err)
}
