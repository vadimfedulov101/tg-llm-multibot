package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

const (
	botHistoryCap  = 256
	botContactsCap = 256
)

type SafeHistory struct {
	mu      sync.RWMutex
	History History
}

func NewSafeHistory(h *History) *SafeHistory {
	return &SafeHistory{
		History: *h,
	}
}

type History map[string]*SafeBotData

// History errors
var (
	ErrGetPathFailed   = errors.New("[history] failed to get history path")
	ErrOpenFailed      = errors.New("[history] failed to open history file")
	ErrReadFailed      = errors.New("[history] failed to read history file")
	ErrWriteFailed     = errors.New("[history] failed to write history file")
	ErrMarshalFailed   = errors.New("[history] failed to marshal history file")
	ErrUnmarshalFailed = errors.New("[history] failed to unmarshal history file")
)

// UNSAFE! Loads history as shared once
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

	log.Println("[memory] history loaded")
	return &history
}

// Saves history
func (sh *SafeHistory) Save(dest string) error {
	// Ensure secure access
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Open/create file
	file, err := os.OpenFile(
		dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644,
	)
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

	log.Println("[memory] history written")
	return nil
}

// Gets safe bot data and its preexistence status
func (sh *SafeHistory) Get(botName string) (*SafeBotData, bool) {
	// Return existing bot data
	if botData, ok := sh.get(botName); ok {
		return botData, true
	}

	// Return new bot data
	return sh.init(botName), false
}

func (sh *SafeHistory) get(botName string) (*SafeBotData, bool) {
	// Ensure secure access
	sh.mu.RLock()
	defer sh.mu.RUnlock()

	botData, ok := sh.History[botName]
	return botData, ok
}

func (sh *SafeHistory) init(botName string) *SafeBotData {
	// Ensure secure access
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Double check initialization after lock release
	if botData, ok := sh.History[botName]; ok {
		return botData
	}

	// Return new bot data
	botData := NewSafeBotData(botHistoryCap, botContactsCap)
	sh.History[botName] = botData
	return botData
}
