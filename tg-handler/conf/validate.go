package conf

import (
	"fmt"
	"log"
	"strings"
)

func mustValidateNumOfS(prompt string, n int, confType string) {
	var err error

	// Handle empty prompt
	if prompt == "" {
		log.Panicf("%v", ErrNoPrompt)
	}

	// Count %s in system prompt
	sNum := strings.Count(prompt, "%s")
	// Detect errors
	if sNum < n {
		err = fmt.Errorf("less than %d %%s", n)
	}
	if sNum > n {
		err = fmt.Errorf("more than %d %%s", n)
	}
	// Return on error
	if err != nil {
		log.Panicf("%v in %s conf: %v", ErrWrongSNum, confType, err)
		log.Panicf("%s", prompt)
	}
}

func mustValidateNumOfD(prompt string, n int, confType string) {
	var err error

	// Handle empty prompt
	if prompt == "" {
		log.Panicf("%v", ErrNoPrompt)
	}

	// Count %s in system prompt
	sNum := strings.Count(prompt, "%d")
	// Detect errors
	if sNum < n {
		err = fmt.Errorf("less than %d %%d", n)
	}
	if sNum > n {
		err = fmt.Errorf("more than %d %%d", n)
	}
	// Return on error
	if err != nil {
		log.Panicf("%v in %s conf: %v", ErrWrongSNum, confType, err)
		log.Panicf("%s", prompt)
	}
}
