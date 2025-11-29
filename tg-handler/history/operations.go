package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
)

// History errors
var (
	ErrGetPathFailed   = errors.New("[history] failed to get history path")
	ErrOpenFailed      = errors.New("[history] failed to open history file")
	ErrReadFailed      = errors.New("[history] failed to read history file")
	ErrWriteFailed     = errors.New("[history] failed to write history file")
	ErrMarshalFailed   = errors.New("[history] failed to marshal history file")
	ErrUnmarshalFailed = errors.New("[history] failed to unmarshal history file")
	ErrCloseFailed     = errors.New("[history] failed to close history file")
)

// Loads history as shared once (non-concurrent)
func MustLoadHistory(source string) *History {
	// Check if source is empty
	if source == "" {
		log.Panicf("%v", ErrGetPathFailed)
	}

	// Start with empty history
	history := make(History)

	// Open/create file
	file, err := os.OpenFile(source, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Panicf("%v: %v", ErrOpenFailed, err)
	}
	defer file.Close()

	// Read JSON data from file
	data, err := io.ReadAll(file)
	if err != nil {
		log.Panicf("%v: %v", ErrReadFailed, err)
	}

	// Decode JSON data to history
	if err := json.Unmarshal(data, &history); err != nil {
		log.Printf("%v: %v", ErrUnmarshalFailed, err)
		log.Printf("[memory] opting to empty history")
	}

	// Close file
	// err = file.Close()
	// if err != nil {
	// log.Panicf("%v: %v", ErrCloseFailed, err)
	// }

	log.Println("[memory] history loaded")
	return &history
}

// Saves history (concurrent)
func (sh *SafeHistory) Save(dest string) error {
	// Ensure secure access
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Open/create file
	file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOpenFailed, err)
	}
	defer file.Close()

	// Encode history to JSON data
	data, err := json.Marshal(sh.History)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMarshalFailed, err)
	}

	// Write JSON data to file
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrWriteFailed, err)
	}

	// Close file
	// err = file.Close()
	// if err != nil {
	// log.Panicf("%v: %v", ErrCloseFailed, err)
	// }

	log.Println("[memory] history written")
	return nil
}

// Gets/initializes safe bot history
func (sh *SafeHistory) Get(botName string) *SafeBotHistory {
	if safeBotHistory := sh.get(botName); safeBotHistory != nil {
		return safeBotHistory
	}
	return sh.init(botName)
}

// Gets/initializes safe chat history
func (sbh *SafeBotHistory) Get(CID int64) *SafeChatHistory {
	if safeChatHistory := sbh.get(CID); safeChatHistory != nil {
		return safeChatHistory
	}
	return sbh.init(CID)
}

// Gets bot history getter
func (sh *SafeHistory) get(botName string) *SafeBotHistory {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.History[botName]
}

// Gets safe chat history
func (sbh *SafeBotHistory) get(CID int64) *SafeChatHistory {
	sbh.mu.RLock()
	defer sbh.mu.RUnlock()
	return sbh.History[CID]
}

// Initializes safe bot history
func (sh *SafeHistory) init(botName string) *SafeBotHistory {
	// Ensure secure access
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Double-check if not initialized
	if safeBotHistory, ok := sh.History[botName]; ok {
		return safeBotHistory
	}

	// Initialize
	botHistory := make(BotHistory)
	safeBotHistory := NewSafeBotHistory(&botHistory)
	sh.History[botName] = safeBotHistory

	return safeBotHistory
}

// Initializes safe chat history
func (sbh *SafeBotHistory) init(CID int64) *SafeChatHistory {
	// Ensure secure access
	sbh.mu.Lock()
	defer sbh.mu.Unlock()

	// Double-check if not initialized
	if safeChatHistory, ok := sbh.History[CID]; ok {
		return safeChatHistory
	}

	// Initialize
	chatHistory := ChatHistory{
		ChatContext: make(ChatContext, 0, 256),
		ReplyChain:  make(ReplyChain, 256),
	}
	safeChatHistory := NewSafeChatHistory(&chatHistory)
	sbh.History[CID] = safeChatHistory

	return safeChatHistory
}
