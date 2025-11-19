package memory

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

// Reconstructs short/long memory from context/reply chain
func GetMemory(sh *SafeChatHistory, lines [2]string, memoryLimit int) *Memory {
	return &Memory{
		ReplyChain:  GetReplyChain(sh, lines, memoryLimit),
		ChatContext: GetChatContext(sh, memoryLimit),
	}
}

// Load history as shared once (non-concurrent)
func LoadHistory(source string) (*History, error) {
	// start with empty history
	history := make(History)

	// Open file (created if needed)
	file, err := os.OpenFile(source, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return &history, fmt.Errorf("Failed to open history file: %w", err)
	}
	defer file.Close()

	// Read JSON data from file
	data, err := io.ReadAll(file)
	if err != nil {
		return &history, fmt.Errorf("Failed to read history file: %w", err)
	}

	// Try to unmarshal if we have data
	if len(data) > 0 {
		// Keep the empty history on fail
		if err := json.Unmarshal(data, &history); err != nil {
			log.Printf("Failed to unmarshal history, using empty: %v", err)
		} else {
			log.Println("History loaded")
		}
	} else {
		log.Println("History created (empty file)")
	}

	return &history, nil
}

// Save history concurrently
func SaveHistory(dest string, sh *SafeHistory) error {
	// Ensure secure access
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Open file
	file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("[OS error] Failed to open history file: %v", err)
	}
	defer file.Close()

	// Encode history to JSON data
	data, err := json.Marshal(sh.History)
	if err != nil {
		return fmt.Errorf("[OS error] Failed to marshal history: %v", err)
	}

	// Write JSON data to file
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("[OS error] Failed to write history data: %v", err)
	}

	log.Println("[OS] History written")
	return nil
}
