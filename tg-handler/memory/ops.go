package memory

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

// Load history as shared once (no concurrency)
func LoadHistory(source string) History {
	var history History

	// Open file (created if needed)
	file, err := os.OpenFile(source, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("[OS error] Failed to open history file: %v", err)
	}
	defer file.Close()

	// Read JSON data from file
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatalf("[OS error] Failed to read history file: %v", err)
	}

	// Decode JSON data to history
	err = json.Unmarshal(data, &history)
	if err != nil {
		history = make(History)
		log.Println("[OS] History created")
	} else {
		log.Println("[OS] History loaded")
	}

	return history
}

// Reconstructs short/long memory from context/reply chain
func GetMemory(sh *SafeChatHistory, lines [2]string, memoryLimit int) *Memory {
	return &Memory{
		ReplyChain:  GetReplyChain(sh, lines, memoryLimit),
		ChatContext: GetChatContext(sh, memoryLimit),
	}
}

// Cleans file history (clean local and save as file)
func CleanFileHistory(sh *SafeHistory, dest string, messageTTL time.Duration) {
	cleanHistory(sh, messageTTL)
	if err := SaveHistory(dest, sh); err != nil {
		log.Printf("Failed to save history: %v", err)
	}
}

// Cleans all lines older than day in every chat history
func cleanHistory(sh *SafeHistory, messageTTL time.Duration) {
	// Get all bot names
	sh.mu.RLock()
	botNames := make([]string, 0, len(sh.History))
	for botName := range sh.History {
		botNames = append(botNames, botName)
	}
	sh.mu.RUnlock()

	currentTime := time.Now()

	// Clean each bot independently
	for _, botName := range botNames {
		safeBotHistory := sh.Get(botName)
		if safeBotHistory != nil {
			cleanBotHistory(safeBotHistory, currentTime, messageTTL)
		}
	}
}

func cleanBotHistory(sbh *SafeBotHistory, currentTime time.Time, messageTTL time.Duration) {
	// Ensure secure access
	sbh.mu.RLock()
	CIDs := make([]int64, 0, len(sbh.History))
	for CID := range sbh.History {
		CIDs = append(CIDs, CID)
	}
	sbh.mu.RUnlock()

	for _, CID := range CIDs {
		safeChatHistory := sbh.Get(CID)
		cleanChatHistory(safeChatHistory, currentTime, messageTTL)

	}
}

func cleanChatHistory(sch *SafeChatHistory, currentTime time.Time, messageTTL time.Duration) {
	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	// Clean up context
	// Create a new slice pointing to the original slice
	filteredContext := sch.History.ChatContext[:0]
	// Range over the original slice reusing its memory for a new slice
	for _, messageEntry := range sch.History.ChatContext {
		if currentTime.Sub(messageEntry.Timestamp) <= messageTTL {
			filteredContext = append(filteredContext, messageEntry)
		}
	}

	// Clean up reply chains
	for line, messageEntry := range sch.History.ReplyChains {
		if currentTime.Sub(messageEntry.Timestamp) > messageTTL {
			delete(sch.History.ReplyChains, line)
		}
	}
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
