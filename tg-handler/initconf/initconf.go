package initconf

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type InitConf struct {
	KeysAPI      []string            `json:"keysAPI"`
	Admins       []string            `json:"admins"`
	Orders       map[string][]string `json:"orders"`
	ConfigPath   string              `json:"config_path"`
	HistoryPath  string              `json:"history_path"`
	MemoryConfig MemoryConfig        `json:"memory_config"`
	Prompts      Prompts             `json:"prompts"`
	CandidateNum int                 `json:"candidate_num"`
}

type MemoryConfig struct {
	Limit           int           `json:"limit"`
	MessageTTL      time.Duration `json:"msg_ttl"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
}

type Prompts struct {
	ResponsePrompt string `json:"response"`
	SelectPrompt   string `json:"select"`
}

func Load(confPath string) (*InitConf, error) {
	var initConf InitConf

	// Read JSON data from file
	data, err := os.ReadFile(confPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read %s: %w", confPath, err)
	}

	// Decode JSON data to InitConf
	err = json.Unmarshal(data, &initConf)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal %s: %w", confPath, err)
	}

	return &initConf, nil
}
