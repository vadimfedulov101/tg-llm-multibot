package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config JSON representation
type Settings struct {
	SystemPrompt string  `json:"system_prompt"`
	Options      Options `json:"options"`
}

type Options struct {
	Temperature   float32 `json:"temperature,omitempty"`
	RepeatPenalty float32 `json:"repeat_penalty,omitempty"`
	TopP          float32 `json:"top_p,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"`
	Seed          int     `json:"seed,omitempty"`
}

// Loads settings
func loadSettings(conf string, chatTitle string) (*Settings, error) {
	var settings Settings

	// Read JSON data from file
	data, err := os.ReadFile(conf)
	if err != nil {
		return nil, fmt.Errorf("Failed to read settings file: %w", err)
	}

	// Decode JSON data to settings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal settings: %w", err)
	}

	// Format system prompt with chat title
	settings.SystemPrompt, err = fmtSystemPrompt(settings.SystemPrompt, chatTitle)
	if err != nil {
		return nil, fmt.Errorf("Failed to format system prompt: %w", err)
	}

	return &settings, nil
}

// Formats system prompt with chat title
func fmtSystemPrompt(systemPrompt string, chatTitle string) (string, error) {
	sNum := strings.Count(systemPrompt, "%s")
	if sNum < 1 {
		return "", errors.New("Less than one %%s (chat title) in system prompt")
	}
	if sNum > 1 {
		return "", errors.New("More than one %%s (chat title) in system prompt")
	}

	return fmt.Sprintf(systemPrompt, chatTitle), nil
}
