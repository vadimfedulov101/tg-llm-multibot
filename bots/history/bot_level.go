package history

import (
	"fmt"
	"strings"
	"sync"

	"tg-handler/carma"
	"tg-handler/tags"
)

// Constants
const (
	botHistoryCap  = 256
	botContactsCap = 256
)

// BOT DATA

// Bot data consists from bot history, chat agnostic bot contacts.
// No pointer swap occures after initialization, no mutex needed.
type BotData struct {
	History  *SafeBotHistory  // Read-only
	Contacts *SafeBotContacts // Read-only
}

func NewBotData() *BotData {
	return &BotData{
		History:  NewSafeBotHistory(),
		Contacts: NewSafeBotContacts(),
	}
}

// BOT HISTORY BRANCH

type SafeBotHistory struct {
	mu      sync.RWMutex
	History BotHistory
}

func NewSafeBotHistory() *SafeBotHistory {
	return &SafeBotHistory{
		History: NewBotHistory(),
	}
}

type BotHistory map[int64]*ChatHistory

func NewBotHistory() BotHistory {
	h := make(BotHistory, botHistoryCap)
	return h
}

// BOT CONTACTS BRANCH

type SafeBotContacts struct {
	mu       sync.RWMutex
	Contacts BotContacts
}

func NewSafeBotContacts() *SafeBotContacts {
	return &SafeBotContacts{
		Contacts: NewBotContacts(),
	}
}

func (sbcs *SafeBotContacts) String() string {
	// Ensure secure access
	sbcs.mu.RLock()
	defer sbcs.mu.RUnlock()

	// Return string
	return sbcs.Contacts.String()
}

type BotContacts map[string]BotContact

func NewBotContacts() BotContacts {
	bc := make(BotContacts, botContactsCap)
	return bc
}

func (bcs BotContacts) String() string {
	var sb strings.Builder

	// Describe contacts
	sb.WriteString("Contacts (users known to you):\n")

	// Present contacts
	if bcs == nil {
		sb.WriteString("<no contacts>")
		return sb.String()
	}
	for userName, contact := range bcs {
		sb.WriteString(
			fmt.Sprintf("user: %s\n%s\n", userName, contact),
		)
	}

	return sb.String()
}

// BOT CONTACT

type BotContact struct {
	Carma carma.Carma
	Tags  tags.Tags
}

func (bc BotContact) String() string {
	return fmt.Sprintf("carma: %d\ntags: %s\n", bc.Carma, bc.Tags)
}

// METHODS

// BOT HISTORY BRANCH

// Gets safe chat history and status
func (sbh *SafeBotHistory) Get(
	cid int64,
	scq *SafeChatQueue, // Preinit for public, nil for private chats
) (*ChatHistory, bool) {
	// Happy path: return existing chat history
	if chatHistory, ok := sbh.get(cid); ok {
		return chatHistory, true
	}

	// Unhappy path: return new chat history
	return sbh.init(cid, scq), false

}

func (sbh *SafeBotHistory) get(cid int64) (*ChatHistory, bool) {
	// Ensure secure access
	sbh.mu.RLock()
	defer sbh.mu.RUnlock()

	chatHistory, ok := sbh.History[cid]
	return chatHistory, ok
}

func (sbh *SafeBotHistory) init(
	cid int64,
	scq *SafeChatQueue, // Preinit for public, nil for private chats
) *ChatHistory {
	// Ensure secure access
	sbh.mu.Lock()
	defer sbh.mu.Unlock()

	// Double check if init after lock release
	if chatHistory, ok := sbh.History[cid]; ok {
		return chatHistory
	}

	// Return new chat history
	chatHistory := NewChatHistory(scq)
	sbh.History[cid] = chatHistory
	return chatHistory
}

// BOT CONTACTS BRANCH

// Gets bot contact
func (sbcs *SafeBotContacts) Get(userName string) BotContact {
	// Ensure secure access
	sbcs.mu.RLock()
	defer sbcs.mu.RUnlock()

	// Return existing bot contact
	if botContact, ok := sbcs.Contacts[userName]; ok {
		return botContact
	}

	// Return new bot contact
	return BotContact{}
}

// Sets bot contact
func (sbcs *SafeBotContacts) Set(
	userName string,
	botContact BotContact,
) {
	// Ensure secure access
	sbcs.mu.Lock()
	defer sbcs.mu.Unlock()

	// Set bot contact
	sbcs.Contacts[userName] = botContact
}
