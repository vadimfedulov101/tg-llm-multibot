package conf

import (
	"encoding/json"
	"errors"
	"log"
	"os"
)

// Bot config errors
var (
	errBReadFailed           = errors.New("[conf] read bot config failed")
	errBUnmarshalFailed      = errors.New("[conf] unmarshal bot config failed")
	errBNegativeCandidateNum = errors.New("[conf] negative candidate number")
)

// Bot config
type BotConf struct {
	Main     MainSettings     `json:"bot_conf"`
	Optional OptionalSettings `json:"options"`
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

// Loads settings or panics
func MustLoadBotConf(confPath string) *BotConf {
	var botConf BotConf

	// Read JSON data from file
	data, err := os.ReadFile(confPath)
	if err != nil {
		log.Fatalf("%v: %v", errBReadFailed, err)
	}

	// Decode JSON data to settings
	err = json.Unmarshal(data, &botConf)
	if err != nil {
		log.Fatalf("%v: %v", errBUnmarshalFailed, err)
	}

	// Validate candidate number or panic
	mustValidateCandidateNum(&botConf)

	return &botConf
}

// Validates candidate num or panics
func mustValidateCandidateNum(conf *BotConf) {
	if conf.Main.CandidateNum < 0 {
		log.Fatal(errBNegativeCandidateNum)
	}
}
