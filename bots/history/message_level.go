package history

import (
	"time"
)

type MessageEntry struct {
	Line      string    `json:"msg"`
	Timestamp time.Time `json:"ts"`
}

func NewMessageEntry(line string) *MessageEntry {
	return &MessageEntry{
		Line:      line,
		Timestamp: time.Now(),
	}
}
