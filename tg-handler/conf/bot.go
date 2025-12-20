package conf

import (
	"encoding/json"
	"errors"
	"log"
	"os"
)

// Bot config
type BotConf struct {
	Main     MainSettings     `json:"main_settings"`
	Optional OptionalSettings `json:"optional_settings"`
}

// Main settings for LLM
type MainSettings struct {
	Role         string `json:"role"`
	CandidateNum int    `json:"candidate_num"`
}

// Optional settings for LLM
type OptionalSettings struct {
	Temperature   float32 `json:"temperature,omitempty"`
	RepeatPenalty float32 `json:"repeat_penalty,omitempty"`
	TopP          float32 `json:"top_p,omitempty"`
	TopK          int     `json:"top_k,omitempty"`
	NumPredict    int     `json:"num_predict,omitempty"`
	Seed          int     `json:"seed,omitempty"`
}

// Bot config errors
var (
	ErrBConfReadFailed = errors.New(
		"[conf] read bot config failed",
	)
	ErrBConfUnmarshalFailed = errors.New(
		"[conf] unmarshal bot config failed",
	)
)

// Loads settings or panics
func MustLoadBotConf(confPath string) *BotConf {
	var botConf BotConf

	// Read JSON data from file
	data, err := os.ReadFile(confPath)
	if err != nil {
		log.Panicf("%v: %v", ErrBConfReadFailed, err)
	}

	// Decode JSON data to settings
	err = json.Unmarshal(data, &botConf)
	if err != nil {
		log.Panicf("%v: %v", ErrBConfUnmarshalFailed, err)
	}

	return &botConf
}
