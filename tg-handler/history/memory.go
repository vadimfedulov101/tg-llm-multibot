package history

import (
	"log"
	"strings"
	"time"

	"tg-handler/messaging"
)

// Memory is constructed from bot's chat history once.
// It consists from:
// 1. Short(-term): Chat context as last N messages
// 2. Long(-term): Reply chain as reply sequence

// Memory representation
type Memory = struct {
	Short []string
	Long  []string
}

func NewMemory(
	sch *SafeChatHistory,
	msgInfo *messaging.MessageInfo,
	memoryLimit int,
) *Memory {
	return &Memory{
		Short: sch.getChatContext(memoryLimit),
		Long:  sch.getReplyChain(msgInfo, memoryLimit),
	}
}

// Gets joined memories as tuple
func GetMemoryStrings(memory *Memory) (string, string) {
	shortS := strings.Join(memory.Short, "\n")
	longS := strings.Join(memory.Long, "\n")
	return shortS, longS
}

// Adds current and previous messages from message msgInfo
// to chat context and reply chain
func (sch *SafeChatHistory) AddTo(msgInfo *messaging.MessageInfo) {
	sch.addToChatContext(msgInfo)
	sch.addToReplyChain(msgInfo)
}

// Adds message to chat context; cleaner eventually cuts to memory limit
func (sch *SafeChatHistory) addToChatContext(msgInfo *messaging.MessageInfo) {
	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	// Add message entry to chat context
	messageEntry := MessageEntry{
		Line:      msgInfo.Line,
		Timestamp: time.Now(),
	}
	sch.History.ChatContext = append(sch.History.ChatContext, messageEntry)

	log.Println("[history] chat context++")
}

// Adds message chain to reply chain; cleaner eventually cleanes expired ones
func (sch *SafeChatHistory) addToReplyChain(msgInfo *messaging.MessageInfo) {
	// Get lines
	lines := sch.getReplyChain(msgInfo, 2)
	// No chain to add
	if len(lines) < 2 {
		return
	}
	// Got chain to add
	var (
		prevLine = lines[0]
		lastLine = lines[1]
	)

	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	// Add message chain to reply chain
	sch.History.ReplyChain[lastLine] = MessageEntry{
		Line:      prevLine,
		Timestamp: time.Now(),
	}

	log.Println("[history] reply chain++")
}

// Extracts chat context from safe chat history
func (sch *SafeChatHistory) getChatContext(memoryLimit int) []string {
	// Ensure secure access
	sch.mu.RLock()
	defer sch.mu.RUnlock()

	// Accumulate lines with memory limit
	lines := make([]string, 0, memoryLimit)
	for i, messageEntry := range sch.History.ChatContext {
		if i+1 > memoryLimit {
			break
		}
		lines = append(lines, messageEntry.Line)
	}

	log.Printf("[history] chat context: %d messages", len(lines))
	return lines
}

// Extracts reply chain from safe chat history
func (sch *SafeChatHistory) getReplyChain(
	msgInfo *messaging.MessageInfo,
	memoryLimit int,
) []string {
	// Ensure secure access
	sch.mu.RLock()
	defer sch.mu.RUnlock()

	// Return one line if got only last one
	lastLine := msgInfo.Line
	if msgInfo.Prev == nil {
		return []string{lastLine}
	}
	// Set previous line
	prevLine := msgInfo.Prev.Line

	// Accumulate lines going backwards via reply chain
	replyChain := []string{lastLine, prevLine}
	for range memoryLimit - 2 {
		lastLine = prevLine
		if messageEntry, ok := sch.History.ReplyChain[lastLine]; ok {
			prevLine = messageEntry.Line
			replyChain = append(replyChain, prevLine)
		} else {
			break
		}
	}

	// Reverse reply chain
	for i, j := 0, len(replyChain)-1; i < j; i, j = i+1, j-1 {
		replyChain[i], replyChain[j] = replyChain[j], replyChain[i]
	}

	log.Printf("[history] reply chain: %d messages", len(replyChain))
	return replyChain
}
