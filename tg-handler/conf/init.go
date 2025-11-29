package conf

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"
)

// Initialization config errors
var (
	ErrReadFailed      = errors.New("[conf] read init config failed")
	ErrUnmarshalFailed = errors.New("[conf] unmarshal init config failed")
)

// Initialization config
type InitConf struct {
	BotsConf     BotsConf     `json:"bots_conf"`
	GenerateConf GenerateConf `json:"generate_conf"`
	PathsConf    PathsConf    `json:"paths_conf"`
	CleanerConf  CleanerConf  `json:"cleaner_conf"`
}

// Bots config
type BotsConf struct {
	KeysAPI []string `json:"keysAPI"`
	Admins  []string `json:"admins"`
	CIDs    []int64  `json:"cids"`
}

// Generate config
type GenerateConf struct {
	Prompts     Prompts `json:"prompts"`
	MemoryLimit int     `json:"memory_limit"`
}

// Paths config
type PathsConf struct {
	History string `json:"history"`
	Bots    string `json:"bots"`
}

// Cleaner config
type CleanerConf struct {
	MessageTTL      Duration `json:"msg_ttl"`
	CleanupInterval Duration `json:"cleanup_interval"`
}

// Prompts common for all bots
type Prompts struct {
	Response string `json:"response"`
	Select   string `json:"select"`
}

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// Loads init config or panics
func MustLoadInitConf(confPath string) *InitConf {
	var initConf InitConf

	// Read JSON data from file
	data, err := os.ReadFile(confPath)
	if err != nil {
		log.Panicf("%v (%s): %v", ErrReadFailed, confPath, err)
	}

	// Decode JSON data to InitConf
	err = json.Unmarshal(data, &initConf)
	if err != nil {
		log.Panicf("%v (%s): %v", ErrUnmarshalFailed, confPath, err)
	}

	// Validate prompts
	mustValidatePrompts(&initConf.GenerateConf.Prompts)

	return &initConf
}

// Validates prompts
func mustValidatePrompts(prompts *Prompts) {
	mustValidateResponsePrompt(prompts.Response)
	mustValidateSelectPrompt(prompts.Select)
}

// Validates response prompt
func mustValidateResponsePrompt(responsePrompt string) {
	mustValidateNumOfS(responsePrompt, 3, "init")
}

// Validates select prompt
func mustValidateSelectPrompt(selectPrompt string) {
	mustValidateNumOfS(selectPrompt, 3, "init")
	mustValidateNumOfD(selectPrompt, 1, "init")
}
