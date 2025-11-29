package api

import (
	"encoding/json"
	"errors"
	"log"
	"os"

	"tg-handler/conf"
)

// Settings for LLM inference
type Settings struct {
	BotConf conf.BotConf `json:"bot_conf"`
	Options Options      `json:"options"`
}

// Options for LLM
type Options struct {
	Temperature   float32 `json:"temperature,omitempty"`
	RepeatPenalty float32 `json:"repeat_penalty,omitempty"`
	TopP          float32 `json:"top_p,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"`
	Seed          int     `json:"seed,omitempty"`
}

// Settings errors
var (
	ErrReadFailed      = errors.New("[api] read settings failed")
	ErrUnmarshalFailed = errors.New("[api] unmarshal settings failed")
)

// Loads settings or panics
func mustLoadSettings(confPath string) *Settings {
	var settings Settings

	// Read JSON data from file
	data, err := os.ReadFile(confPath)
	if err != nil {
		log.Panicf("%v: %v", ErrReadFailed, err)
	}

	// Decode JSON data to settings
	err = json.Unmarshal(data, &settings)
	if err != nil {
		log.Panicf("%v: %v", ErrUnmarshalFailed, err)
	}

	// Validate system prompt
	systemPrompt := settings.BotConf.SystemPrompt
	conf.MustValidateSystemPrompt(systemPrompt)

	return &settings
}
