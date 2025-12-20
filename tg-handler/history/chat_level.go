package history

import (
	"log"
	"sync"
)

// messaging.MessageInfo abstraction
type LineChain interface {
	lineProvider
	prevLineProvider
}

type lineProvider interface {
	Line() string
}

type prevLineProvider interface {
	PrevLine() string
}

// (SAFE) CHAT HISTORY

type SafeChatHistory struct {
	mu      sync.RWMutex
	History ChatHistory
}

func NewSafeChatHistory(
	chatQueueSize int,
	replyChainsSize int,
) *SafeChatHistory {
	return &SafeChatHistory{
		History: *NewChatHistory(chatQueueSize, replyChainsSize),
	}
}

type ChatHistory struct {
	ChatQueue   SafeChatQueue
	ReplyChains SafeReplyChains
}

func NewChatHistory(
	chatQueueSize int,
	replyChainsSize int,
) *ChatHistory {
	return &ChatHistory{
		ChatQueue:   *NewSafeChatQueue(chatQueueSize),
		ReplyChains: *NewSafeReplyChains(replyChainsSize),
	}
}

// CHAT QUEUE BRANCH

type SafeChatQueue struct {
	mu        sync.Mutex
	ChatQueue ChatQueue
}

func NewSafeChatQueue(size int) *SafeChatQueue {
	return &SafeChatQueue{
		ChatQueue: *NewChatQueue(size),
	}

}

type ChatQueue []MessageEntry

func NewChatQueue(size int) *ChatQueue {
	c := make(ChatQueue, 0, size)
	return &c
}

// REPLY CHAINS BRANCH

type SafeReplyChains struct {
	mu          sync.Mutex
	ReplyChains ReplyChains
}

func NewSafeReplyChains(size int) *SafeReplyChains {
	return &SafeReplyChains{
		ReplyChains: *NewReplyChains(size),
	}

}

type ReplyChains map[string]MessageEntry

func NewReplyChains(size int) *ReplyChains {
	r := make(ReplyChains, size)
	return &r
}

// METHODS

// Unpacks chat queue and reply chains from safe chat history
func (sch *SafeChatHistory) Unpack() (*SafeChatQueue, *SafeReplyChains) {
	// Ensure secure access
	sch.mu.RLock()
	defer sch.mu.RUnlock()

	// Get history and its content
	history := &sch.History
	return &history.ChatQueue, &history.ReplyChains
}

// Adds message to queue and chains
func (sch *SafeChatHistory) AddTo(lc LineChain) {
	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	// Unpack safe chat history
	chatQueue, replyChain := sch.Unpack()

	chatQueue.AddTo(lc)
	replyChain.AddTo(lc)
}

// Add to chat queue
func (scq *SafeChatQueue) AddTo(lc LineChain) {
	scq.AppendLine(lc.Line())
	log.Println("[history] chat queue++")
}

// Add to reply chains
func (src *SafeReplyChains) AddTo(lc LineChain) {
	lines := src.GetLines(lc.PrevLine(), lc.Line(), 2)
	src.SetLines(lines)

	log.Println("[history] reply chains++")
}

// Appends line to safe chat queue
func (scq *SafeChatQueue) AppendLine(line string) {
	// Ensure secure access
	scq.mu.Lock()
	defer scq.mu.Unlock()

	// Get message entry
	messageEntry := *NewMessageEntry(line)

	// Append message entry to chat queue
	scq.ChatQueue = append(scq.ChatQueue, messageEntry)
}

// Gets lines from safe chat queue
func (scq *SafeChatQueue) GetLines(limit int) []string {
	lines := make([]string, 0, limit)

	// Ensure secure access
	scq.mu.Lock()
	defer scq.mu.Unlock()

	// Get chat queue
	chatQueue := scq.ChatQueue

	// Accumulate lines with memory limit
	for i, messageEntry := range chatQueue {
		if i+1 > limit {
			break
		}
		lines = append(lines, messageEntry.Line)
	}

	log.Printf(
		"[history] chat queue: %d message lines", len(lines),
	)
	return lines
}

// Sets lines in safe reply chains
func (src *SafeReplyChains) SetLines(lines []string) {
	// Incomplete or too long chain, no set
	if len(lines) != 2 {
		return
	}

	// Ensure secure access
	src.mu.Lock()
	defer src.mu.Unlock()

	// Get reply chains
	replyChains := src.ReplyChains

	// Set lines
	var (
		prevLine = lines[0]
		lastLine = lines[1]
	)

	// Set chain
	replyChains[lastLine] = *NewMessageEntry(prevLine)
}

// Gets lines from safe reply chains with limit
func (src *SafeReplyChains) GetLines(
	prevLine string,
	lastLine string,
	limit int,
) []string {
	// Incomplete chain, no unroll
	if prevLine == "" {
		return []string{lastLine}
	}
	// Complete chain, proceed to unroll
	replyChain := []string{lastLine, prevLine}

	// Ensure secure access
	src.mu.Lock()
	defer src.mu.Unlock()

	// Get reply chains
	replyChains := src.ReplyChains

	// Accumulate lines unrolling reply chain backwards
	// up to memory limit
	for range limit - 2 {
		lastLine = prevLine
		if messageEntry, ok := replyChains[lastLine]; ok {
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

	log.Printf(
		"[history] reply chain: %d message lines", len(replyChain),
	)
	return replyChain
}
