package memory

import (
	"sync"
	"time"
)

// We have a common history representation across all bots and chats.
// Each level has its own mutex to guard its level of history.
// Only safe GetInit methods are exposed as public, structured for maximum
// performance.

// Message interface
type IMessage interface {
	GetText() string
	GetSender() string
	GetOrder() string
}

// Memory constructed from history
type Memory = struct {
	ReplyChain  []string
	ChatContext []string
}

// History types
type (
	History     = map[string]*SafeBotHistory
	BotHistory  = map[int64]*SafeChatHistory
	ChatHistory = struct {
		ReplyChains map[string]MessageEntry
		ChatContext []MessageEntry
	}
)

// Message type
type MessageEntry struct {
	Line      string    `json:"msg"`
	Timestamp time.Time `json:"ts"`
}

// Safe history types (same mutex to reuse)
type SafeHistory struct {
	History History
	mu      sync.RWMutex
}
type SafeBotHistory struct {
	History BotHistory
	mu      sync.RWMutex
}
type SafeChatHistory struct {
	History ChatHistory
	mu      sync.RWMutex
}

// Memory constructor
func NewMemory(r []string, c []string) *Memory {
	return &Memory{
		ReplyChain:  r,
		ChatContext: c,
	}
}

// Safe history constructor for main
func NewSafeHistory(h *History) *SafeHistory {
	return &SafeHistory{
		History: *h,
	}
}

// Init bot history getter
func (sh *SafeHistory) Get(botName string) *SafeBotHistory {
	if safeBotHistory := sh.get(botName); safeBotHistory != nil {
		return safeBotHistory
	}
	return sh.init(botName)
}

// Init chat history getter
func (sbh *SafeBotHistory) Get(CID int64) *SafeChatHistory {
	if safeChatHistory := sbh.get(CID); safeChatHistory != nil {
		return safeChatHistory
	}
	return sbh.init(CID)
}

// Bot history getter
func (sh *SafeHistory) get(botName string) *SafeBotHistory {
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	return sh.History[botName]
}

// Chat history getter
func (sbh *SafeBotHistory) get(CID int64) *SafeChatHistory {
	sbh.mu.RLock()
	defer sbh.mu.RUnlock()
	return sbh.History[CID]
}

// Bot history initializer
func (sh *SafeHistory) init(botName string) *SafeBotHistory {
	// Ensure secure access
	sh.mu.Lock()
	defer sh.mu.Unlock()

	// Duoble-check if not initialized
	if safeBotHistory, ok := sh.History[botName]; ok {
		return safeBotHistory
	}

	// Actually initialize
	safeBotHistory := &SafeBotHistory{
		History: make(BotHistory),
	}
	sh.History[botName] = safeBotHistory

	// Return initialized
	return safeBotHistory
}

// Chat history initializer
func (sbh *SafeBotHistory) init(CID int64) *SafeChatHistory {
	// Ensure secure access
	sbh.mu.Lock()
	defer sbh.mu.Unlock()

	// Duoble-check if not initialized
	if safeChatHistory, ok := sbh.History[CID]; ok {
		return safeChatHistory
	}

	// Actually initialize
	safeChatHistory := &SafeChatHistory{
		History: ChatHistory{
			ReplyChains: make(map[string]MessageEntry),
			ChatContext: make([]MessageEntry, 0),
		},
	}
	sbh.History[CID] = safeChatHistory

	// Return initialized
	return safeChatHistory
}
