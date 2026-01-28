package history

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"google.golang.org/protobuf/proto"

	"tg-handler/history/pb"
	"tg-handler/logging"
)

// History errors
var (
	errGetPathFailed   = errors.New("failed to get path")
	errReadFailed      = errors.New("failed to read file")
	errUnmarshalFailed = errors.New("failed to unmarshal file")
)

// History consists from bot histories, bot-agnostic shared queues.
// No pointer swap occures after initialization, no mutex needed.
type History struct {
	Bots             *SafeBotsHistory // Read-only (secured inside)
	SharedChatQueues SharedChatQueues // Read-only
}

func NewHistory(cids []int64) *History {
	return &History{
		Bots:             NewSafeBotsHistory(),
		SharedChatQueues: NewSharedChatQueues(cids),
	}
}

// Safe bot history consists from history read/written concurrently
// by bots, cleaner, so mutex needed.
type SafeBotsHistory struct {
	mu      sync.RWMutex
	History BotsHistory
}

func NewSafeBotsHistory() *SafeBotsHistory {
	return &SafeBotsHistory{
		History: make(BotsHistory),
	}
}

// Bot data storage
type BotsHistory map[string]*BotData

// Shared chat queues for all allowed public chats,
// implicitly used to set chat queue on chat level if public
// to avoid memory duplication and preserve simplicity for bots.
type SharedChatQueues map[int64]*SafeChatQueue

func NewSharedChatQueues(cids []int64) SharedChatQueues {
	cq := make(SharedChatQueues, len(cids))
	for _, cid := range cids {
		cq[cid] = NewSafeChatQueue(true)
	}
	return cq
}

// UNSAFE! Loads history or panics
func MustLoadHistory(
	source string,
	cids []int64,
	logger *logging.Logger,
) *History {
	const errMsg = "failed to load history"

	// Check if source is empty
	if source == "" {
		logger.Panic(errMsg, logging.Err(errGetPathFailed))
	}

	// Try to read file
	data, err := os.ReadFile(source)
	if os.IsNotExist(err) {
		// Return new if file doesn't exist
		return NewHistory(cids)
	} else if err != nil {
		logger.Panic(
			errMsg,
			logging.Err(fmt.Errorf("%w: %v", errReadFailed, err)),
		)
	}

	// Unmarshal
	var protoRoot pb.RootHistory
	if err := proto.Unmarshal(data, &protoRoot); err != nil {
		logger.Error(
			errMsg,
			logging.Err(
				fmt.Errorf("%w: v", errUnmarshalFailed, err),
			),
		)

		logger.Info("opting to empty history")
		return NewHistory(cids)
	}

	// Convert back to internal structure
	history := fromProto(&protoRoot, cids)

	logger.Info("history loaded")
	return history
}

// Gets bot data
func (sbh *SafeBotsHistory) Get(botName string) *BotData {
	// Happy path: Return existing bot data
	if botData, ok := sbh.get(botName); ok {
		return botData
	}

	// Unhappy path: Return new bot data
	return sbh.init(botName)
}

// Return existing bot data with status
func (sbh *SafeBotsHistory) get(botName string) (*BotData, bool) {
	// Ensure secure access
	sbh.mu.RLock()
	defer sbh.mu.RUnlock()

	botData, ok := sbh.History[botName]
	return botData, ok
}

// Create new bot data
func (sbh *SafeBotsHistory) init(botName string) *BotData {
	// Ensure secure access
	sbh.mu.Lock()
	defer sbh.mu.Unlock()

	// Double check if init after lock release
	if botData, ok := sbh.History[botName]; ok {
		return botData
	}

	// Return new bot data
	botData := NewBotData()
	sbh.History[botName] = botData
	return botData
}

// Locks history in cascade
func (h *History) lock() {
	var (
		scqs = h.SharedChatQueues
		bots = h.Bots
	)

	// Firstly lock SHARED chat queues
	for _, scq := range scqs {
		scq.mu.Lock()
	}

	// Secondly lock LOCAL chat queues and reply chains
	bots.mu.Lock()
	for _, botData := range bots.History {
		var (
			history  = botData.History
			contacts = botData.Contacts
		)
		history.mu.Lock()
		contacts.mu.Lock()

		for _, sch := range history.History {
			var (
				chatQueue   = sch.ChatQueue
				replyChains = sch.ReplyChains
			)

			// Lock chat queue if local
			if !chatQueue.IsShared {
				chatQueue.mu.Lock()
			}

			// Lock reply chains (always local)
			replyChains.mu.Lock()
		}
	}
}

// Unlocks history in cascade
func (h *History) unlock() {
	var (
		scqs = h.SharedChatQueues
		bots = h.Bots
	)

	// Firstly lock SHARED chat queues
	for _, scq := range scqs {
		scq.mu.Unlock()
	}

	// Secondly lock LOCAL chat queues and reply chains
	bots.mu.Unlock()
	for _, botData := range bots.History {
		var (
			history  = botData.History
			contacts = botData.Contacts
		)
		history.mu.Unlock()
		contacts.mu.Unlock()

		for _, sch := range history.History {
			var (
				chatQueue   = sch.ChatQueue
				replyChains = sch.ReplyChains
			)

			// Lock chat queue if local
			if !chatQueue.IsShared {
				chatQueue.mu.Unlock()
			}

			// Lock reply chains (always local)
			replyChains.mu.Unlock()
		}
	}
}
