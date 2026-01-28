package history

import (
	"fmt"
	"sync"
	"tg-handler/logging"
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

// Chat level errors
const replyChainLenToAdd = 2

var (
	errReplyChainTooLong = fmt.Errorf(
		"reply chain longer than %d to be added",
		replyChainLenToAdd,
	)
	errReplyChainTooShort = fmt.Errorf(
		"reply chain shorter than %d to be added",
		replyChainLenToAdd,
	)
)

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

// Gets queue from safe chat queue with limit
func (scq *SafeChatQueue) Get(
	lim int, logger *logging.Logger,
) []string {
	// Ensure secure access
	scq.mu.RLock()
	defer scq.mu.RUnlock()

	// Call private getter
	return scq.ChatQueue.get(lim, logger)
}

// Gets chain from reply chains with limit
func (src *SafeReplyChains) Get(
	lc LineChain, lim int, logger *logging.Logger,
) []string {
	// Ensure secure access
	src.mu.RLock()
	defer src.mu.RUnlock()

	// Call private getter
	return src.ReplyChains.get(lc, lim, logger)
}

// Adds data to chat queue and reply chains
func (ch *ChatHistory) AddToBoth(
	lc LineChain, logger *logging.Logger,
) {
	ch.ChatQueue.add(lc, logger)
	ch.ReplyChains.add(lc, logger)
}

// Adds data to chat queue
func (ch *ChatHistory) AddToChatQueue(
	lc LineChain, logger *logging.Logger,
) {
	ch.ChatQueue.add(lc, logger)
}

// Adds message line to chat queue
func (scq *SafeChatQueue) add(
	lc LineChain, logger *logging.Logger,
) {
	// Ensure secure access
	scq.mu.Lock()
	defer scq.mu.Unlock()

	// Call private setter
	scq.ChatQueue.add(lc, scq.IsShared, logger)
}

// Adds message lines to reply chains
func (src *SafeReplyChains) add(
	lc LineChain,
	logger *logging.Logger,
) {

	// Ensure secure access
	src.mu.Lock()
	defer src.mu.Unlock()

	// Call private setter
	src.ReplyChains.add(lc, logger)
}

// Adds message line to chat queue
func (cq *ChatQueue) add(
	lc LineChain, isShared bool, logger *logging.Logger,
) {
	var line = lc.Line()

	// Check if line added to shared queue
	if isShared {
		lastLine := cq.get(1, logger)[0]
		if lastLine == line {
			logger.Debug("line skipped as added")
			return
		}
	}

	// Add line
	*cq = append(*cq, *NewMessageEntry(line))

	// Log line added
	logger = logger.With(logging.LastLine(line))
	logger.Debug("line added")
}

// Adds message chain to reply chains
func (rc ReplyChains) add(
	lc LineChain, logger *logging.Logger,
) {
	// Get chain
	chain := rc.get(lc, replyChainLenToAdd, logger)

	// Check chain length
	logger = logger.With(logging.ReplyChainLen(len(chain)))
	const ErrMsg = "failed to add reply chain"
	if len(chain) < replyChainLenToAdd {
		logger.Error(ErrMsg, logging.Err(errReplyChainTooShort))
		return
	}
	if len(chain) > replyChainLenToAdd {
		logger.Error(ErrMsg, logging.Err(errReplyChainTooLong))
		return
	}

	// Add chain
	var (
		prevLine = chain[0]
		lastLine = chain[1]
	)
	rc[lastLine] = *NewMessageEntry(prevLine)

	// Log chain added
	logger = logger.With(logging.PrevLine(prevLine))
	logger = logger.With(logging.LastLine(lastLine))
	logger.Debug("reply chain added")
}

// Gets lines from chat queue with limit
func (cq ChatQueue) get(lim int, logger *logging.Logger) []string {
	queue := make([]string, 0, lim)

	// DO WE NEED A CHECK HERE LIKE
	// if len(cq) < 1 { return } ?
	// because if the rest of the code
	// works well with zero len, we don't

	// Shift by limit if exceeded
	shift := min(len(cq), lim)
	// Get start index by shifting
	start := len(cq) - shift

	// Accumulate lines
	for _, msg := range cq[start:] {
		queue = append(queue, msg.Line)
	}

	// Log getting chat queue
	logger = logger.With(logging.ChatQueueLen(len(queue)))
	logger.Debug("got chat queue")
	return queue
}

// Gets reply chain with limit
func (rc ReplyChains) get(
	lc LineChain, lim int, logger *logging.Logger,
) []string {
	var (
		prevLine = lc.PrevLine()
		lastLine = lc.Line()
	)

	chain := []string{lastLine}

	// Handle incomplete reply chain
	if prevLine == "" {
		logger.Debug("incomplete reply chain, no unroll")
		return chain
	}
	logger.Debug("complete reply chain, proceed to unroll")

	// Accumulate lines unrolling reply chain backwards up to limit
	chain = append(chain, prevLine)
	for range lim - 2 {
		lastLine = prevLine
		if msg, ok := rc[lastLine]; ok {
			prevLine = msg.Line
			chain = append(chain, prevLine)
		} else {
			break
		}
	}

	// Reverse reply chain
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	// Log getting reply chain
	logger = logger.With(logging.ReplyChainLen(len(chain)))
	logger.Debug("got reply chain")
	return chain
}
