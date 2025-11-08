package memory

import (
	"log"
	"time"
)

// Extracts reply chain from safe chat history
func GetReplyChain(sch *SafeChatHistory, lines [2]string, memoryLimit int) []string {
	// Ensure secure access
	sch.mu.RLock()
	defer sch.mu.RUnlock()

	// Format lines, return if got only last one
	lastLine := lines[0]
	prevLine := lines[1]
	if prevLine == "" {
		return []string{lastLine}
	}

	// Accumulate lines going backwards via reply chain
	replyChain := []string{lastLine, prevLine}
	lastLine = prevLine
	for range memoryLimit - 2 {
		if messageEntry, ok := sch.History.ReplyChains[lastLine]; ok {
			prevLine = messageEntry.Line
			replyChain = append(replyChain, prevLine)
			lastLine = prevLine
		} else {
			break
		}
	}

	// Reverse reply chain
	for i, j := 0, len(replyChain)-1; i < j; i, j = i+1, j-1 {
		replyChain[i], replyChain[j] = replyChain[j], replyChain[i]
	}

	log.Printf("Long-term memory: %d messages", len(replyChain))
	return replyChain
}

// Adds messages to reply chain in safe chat history
func AddToReplyChain(sch *SafeChatHistory, ms [2]IMessage) [2]string {
	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	// Format lines, return if got only last one
	lastLine := Format(ms[0])
	prevLine := Format(ms[1])
	if prevLine == "" {
		return [2]string{lastLine, prevLine}
	}

	// Record lines connection
	sch.History.ReplyChains[lastLine] = MessageEntry{
		Line:      prevLine,
		Timestamp: time.Now(),
	}

	// Return in the same order
	return [2]string{lastLine, prevLine}
}

// Adds messages to reply chain in safe chat history
func AddToReplyChainReuse(sch *SafeChatHistory, m IMessage, prevLine string) [2]string {
	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	// Format line
	lastLine := Format(m)

	// Record lines connection
	sch.History.ReplyChains[lastLine] = MessageEntry{
		Line:      prevLine,
		Timestamp: time.Now(),
	}

	// Return in the same order
	return [2]string{lastLine, prevLine}
}
