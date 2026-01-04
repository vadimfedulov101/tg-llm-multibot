package history

import (
	"log"
	"sync"
)

const (
	chatQueueCap   = 256
	replyChainsCap = 256
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

// CHAT HISTORY

// Chat history consists from chat queue and reply chains.
// No pointer swap occures after initialization, no mutex needed.
type ChatHistory struct {
	ChatQueue   *SafeChatQueue   // Read-only
	ReplyChains *SafeReplyChains // Read-only
}

// Constructs chat history with
// shared queue for public chats, local queue for private chats.
func NewChatHistory(scq *SafeChatQueue) *ChatHistory {
	// If no safe queue, create it as local
	if scq == nil {
		scq = NewSafeChatQueue(false)
	}
	// Set queue as local or shared
	return &ChatHistory{
		ChatQueue:   scq,
		ReplyChains: NewSafeReplyChains(),
	}
}

// CHAT QUEUE BRANCH

type SafeChatQueue struct {
	mu        sync.RWMutex
	ChatQueue ChatQueue

	IsShared bool
}

// Constructs safe chat queue
// shared for public chats, local for private.
func NewSafeChatQueue(isShared bool) *SafeChatQueue {
	return &SafeChatQueue{
		ChatQueue: NewChatQueue(),
		IsShared:  isShared,
	}
}

type ChatQueue []MessageEntry

func NewChatQueue() ChatQueue {
	c := make(ChatQueue, 0, chatQueueCap)
	return c
}

// REPLY CHAINS BRANCH

type SafeReplyChains struct {
	mu          sync.RWMutex
	ReplyChains ReplyChains
}

func NewSafeReplyChains() *SafeReplyChains {
	return &SafeReplyChains{
		ReplyChains: NewReplyChains(),
	}

}

type ReplyChains map[string]MessageEntry

func NewReplyChains() ReplyChains {
	r := make(ReplyChains, replyChainsCap)
	return r
}

// METHODS

// Adds message to queue
func (ch *ChatHistory) AddToChatQueue(lc LineChain) {
	ch.ChatQueue.add(lc)
}

// Adds message to queue and chains
func (ch *ChatHistory) AddToBoth(lc LineChain) {
	ch.ChatQueue.add(lc)
	ch.ReplyChains.add(lc)
}

// Gets lines from safe chat queue with limit
func (scq *SafeChatQueue) GetLines(lim int) []string {
	// Ensure secure access
	scq.mu.RLock()
	defer scq.mu.RUnlock()

	// Call private getter securely
	return scq.ChatQueue.getLines(lim)
}

// Gets lines from reply chains with limit
func (src *SafeReplyChains) GetLines(
	prevLine string,
	lastLine string,
	lim int,
) []string {
	// Ensure secure access
	src.mu.RLock()
	defer src.mu.RUnlock()

	// Call private getter securely
	return src.ReplyChains.getLines(prevLine, lastLine, lim)
}

// Adds message line to chat queue
func (scq *SafeChatQueue) add(lc LineChain) {
	// Ensure secure access
	scq.mu.Lock()
	defer scq.mu.Unlock()

	// For shared chat queue check if line has been added
	var (
		isShared = scq.IsShared
		cq       = &scq.ChatQueue
	)
	if isShared && len(*cq) > 0 {
		lastMsg := (*cq)[len(*cq)-1]
		if lastMsg.Line == lc.Line() {
			return
		}
	}

	cq.appendLine(lc.Line())
	log.Println("[history] chat queue++")
}

// Adds message line to reply chains atomically
func (src *SafeReplyChains) add(lc LineChain) {
	// Ensure secure access
	src.mu.Lock()
	defer src.mu.Unlock()

	// Get reply chains
	rc := src.ReplyChains

	// Get chain and set it
	lines := rc.getLines(lc.PrevLine(), lc.Line(), 2)
	if ok := rc.setLines(lines); ok {
		log.Println("[history] reply chains++")
	}
}

// Appends message line to chat queue
func (cq *ChatQueue) appendLine(line string) {
	*cq = append(*cq, *NewMessageEntry(line))
}

// Gets lines from chat queue with limit
func (cq ChatQueue) getLines(lim int) []string {
	lines := make([]string, 0, lim)

	// Shift by limit if exceeded
	shift := min(len(cq), lim)
	// Get start index by shifting
	start := len(cq) - shift

	// Accumulate lines
	for _, msg := range cq[start:] {
		lines = append(lines, msg.Line)
	}

	log.Printf("[history] chat queue: %d lines", len(lines))
	return lines
}

// Gets lines from reply chains with limit
func (rc ReplyChains) getLines(
	prevLine string,
	lastLine string,
	lim int,
) []string {
	// Incomplete chain, no unroll
	if prevLine == "" {
		return []string{lastLine}
	}
	// Complete chain, proceed to unroll
	replyChain := []string{lastLine, prevLine}

	// Accumulate lines unrolling reply chain backwards up to limit
	for range lim - 2 {
		lastLine = prevLine
		if msg, ok := rc[lastLine]; ok {
			prevLine = msg.Line
			replyChain = append(replyChain, prevLine)
		} else {
			break
		}
	}

	// Reverse reply chain
	for i, j := 0, len(replyChain)-1; i < j; i, j = i+1, j-1 {
		replyChain[i], replyChain[j] = replyChain[j], replyChain[i]
	}

	log.Printf("[history] reply chain: %d lines", len(replyChain))
	return replyChain
}

// Sets message lines in safe reply chains
func (rc ReplyChains) setLines(lines []string) bool {
	// Incomplete or too long chain, no set
	if len(lines) != 2 {
		return false
	}

	// Set lines
	var (
		prevLine = lines[0]
		lastLine = lines[1]
	)

	// Set chain
	rc[lastLine] = *NewMessageEntry(prevLine)

	return true
}
