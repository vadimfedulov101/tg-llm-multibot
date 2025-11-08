package initconf

import (
	"encoding/json"
	"log"
	"os"
)

// Path, structure and loader for Initial Config
const InitConf = "./confs/init.json"

type InitJSON struct {
	KeysAPI     []string            `json:"keysAPI"`
	Admins      []string            `json:"admins"`
	Orders      map[string][]string `json:"orders"`
	ConfigPath  string              `json:"config_path"`
	HistoryPath string              `json:"history_path"`
	MemoryLimit int                 `json:"memory_limit"`
}

func Load(config string) *InitJSON {
	var initJSON InitJSON

	// Read JSON data from file
	data, err := os.ReadFile(config)
	if err != nil {
		log.Fatalf("Failed to read %s: %v", InitConf, err)
	}

	// Decode JSON data to InitJSON
	err = json.Unmarshal(data, &initJSON)
	if err != nil {
		log.Fatalf("Failed to unmarshal %s: %v", InitConf, err)
	}

	return &initJSON
}
