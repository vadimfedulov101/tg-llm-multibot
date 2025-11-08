package memory

import (
	"log"
	"time"
)

// Extract chat context from safe chat history
func GetChatContext(sch *SafeChatHistory, memoryLimit int) []string {
	// Ensure secure access
	sch.mu.RLock()
	defer sch.mu.RUnlock()

	// Accumulate all lines in chronological order
	lines := make([]string, 0, memoryLimit)
	for _, messageEntry := range sch.History.ChatContext {
		lines = append(lines, messageEntry.Line)
	}

	log.Printf("Short-term memory: %d messages", len(lines))
	return lines
}

// Add messages to chat context in safe chat history
func AddToChatContext(sch *SafeChatHistory, m IMessage, memoryLimit int) string {
	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	// Cut down to the memory limit
	if len(sch.History.ChatContext) >= memoryLimit {
		sch.History.ChatContext = sch.History.ChatContext[:memoryLimit]
	}

	// Format line
	line := Format(m)

	// Record formatted line into context
	sch.History.ChatContext = append(sch.History.ChatContext, MessageEntry{
		Line:      line,
		Timestamp: time.Now(),
	})

	return line
}
