package translator

import (
	"errors"
	"fmt"

	"github.com/bregydoc/gtranslate"
)

var (
	errMsgTranslationFailed = errors.New(
		"failed to translate from English",
	)
)

// Translates message to language based on its tag
func Translate(msg string) (string, error) {
	msg, err := gtranslate.TranslateWithParams(msg, gtranslate.TranslationParams{
		From: "en",
		To:   "eo",
	})
	if err != nil {
		return "", fmt.Errorf("%w: %v", errMsgTranslationFailed, err)
	}

	return msg, nil
}
