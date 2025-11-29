package history

import (
	"sync"
	"time"
)

// History is common across all bots and chats.
// Each layer is guarded by separate mutex.
// Extra funcs are exposed in operations.go and memory.go

// LEVEL 1: Safe history
type SafeHistory struct {
	mu      sync.RWMutex
	History History
}

func NewSafeHistory(h *History) *SafeHistory {
	return &SafeHistory{
		History: *h,
	}
}

type History map[string]*SafeBotHistory

// LEVEL 2: Safe bot history
type SafeBotHistory struct {
	mu      sync.RWMutex
	History BotHistory
}

func NewSafeBotHistory(h *BotHistory) *SafeBotHistory {
	return &SafeBotHistory{
		History: *h,
	}
}

type BotHistory map[int64]*SafeChatHistory

// LEVEL 3: Safe chat history
type SafeChatHistory struct {
	mu      sync.RWMutex
	History ChatHistory
}

func NewSafeChatHistory(h *ChatHistory) *SafeChatHistory {
	return &SafeChatHistory{
		History: *h,
	}
}

type ChatHistory struct {
	ChatContext ChatContext
	ReplyChain  ReplyChain
}

// LEVEL 4: Memory representation
type (
	ChatContext []MessageEntry
	ReplyChain  map[string]MessageEntry
)

// LEVEL 5: Message representation
type MessageEntry struct {
	Line      string    `json:"msg"`
	Timestamp time.Time `json:"ts"`
}
