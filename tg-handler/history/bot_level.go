package history

import (
	"fmt"
	"strings"
	"sync"
)

const (
	botHistoryCap  = 256
	botContactsCap = 256
)

// (SAFE) BOT DATA

type SafeBotData struct {
	mu   sync.RWMutex
	Data BotData
}

func NewSafeBotData() *SafeBotData {
	return &SafeBotData{
		Data: *NewBotData(),
	}
}

type BotData struct {
	History  SafeBotHistory
	Contacts SafeBotContacts
}

func NewBotData() *BotData {
	return &BotData{
		History:  *NewSafeBotHistory(),
		Contacts: *NewSafeBotContacts(),
	}
}

// SAFE BOT DATA

func (sbd *SafeBotData) Get() (*SafeBotHistory, *SafeBotContacts) {
	// Ensure secure access
	sbd.mu.RLock()
	defer sbd.mu.RUnlock()

	data := &sbd.Data
	return &data.History, &data.Contacts
}

// BOT HISTORY BRANCH

type SafeBotHistory struct {
	mu      sync.RWMutex
	History BotHistory
}

func NewSafeBotHistory() *SafeBotHistory {
	return &SafeBotHistory{
		History: *NewBotHistory(),
	}
}

type BotHistory map[int64]*SafeChatHistory

func NewBotHistory() *BotHistory {
	h := make(BotHistory, botHistoryCap)
	return &h
}

// BOT CONTACTS BRANCH

type SafeBotContacts struct {
	mu       sync.RWMutex
	Contacts BotContacts
}

func NewSafeBotContacts() *SafeBotContacts {
	return &SafeBotContacts{
		Contacts: *NewBotContacts(),
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

func NewBotContacts() *BotContacts {
	bc := make(BotContacts, botContactsCap)
	return &bc
}

func (bcs BotContacts) String() string {
	var sb strings.Builder

	if bcs == nil {
		return "<no contacts>"
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
	Carma int
	Note  string
}

func (bc BotContact) String() string {
	return fmt.Sprintf(
		"carma: %d\nnote:\n%s\n\n", bc.Carma, bc.Note,
	)
}

// METHODS

// BOT HISTORY BRANCH

// Gets safe chat history and preexistence status
func (sbh *SafeBotHistory) Get(cid int64) (*SafeChatHistory, bool) {
	// Return existing chat history
	if chatHistory, ok := sbh.get(cid); ok {
		return chatHistory, true
	}

	// Return new chat history
	return sbh.init(cid), false

}

func (sbh *SafeBotHistory) get(cid int64) (*SafeChatHistory, bool) {
	// Ensure secure access
	sbh.mu.RLock()
	defer sbh.mu.RUnlock()

	chatHistory, ok := sbh.History[cid]
	return chatHistory, ok
}

func (sbh *SafeBotHistory) init(cid int64) *SafeChatHistory {
	// Ensure secure access
	sbh.mu.Lock()
	defer sbh.mu.Unlock()

	// No double check of initialization after lock release
	// as there is one goroutine per bot & cleaner skips new data
	// if chatHistory, ok := sbh.History[cid]; ok {
	//	return chatHistory
	// }

	// Return new chat histroy
	chatHistory := NewSafeChatHistory()
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
