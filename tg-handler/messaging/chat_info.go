package messaging

import (
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/history"
)

type ChatInfo struct {
	ID        int64
	Title     string
	History   *history.ChatHistory
	IsAllowed bool
	IsPrivate bool
	LastMsg   *MessageInfo
}

// Constructs chat info by following bot procedure
// on how to validate chat ID. Reuses chat queues
// for public chats.
func NewChatInfo(
	m *MessageInfo,
	sbh *history.SafeBotHistory,
	shared history.SharedChatQueues,
	validateChatID func(int64) bool,
) *ChatInfo {
	// Get message vars
	var (
		chat        = m.Chat
		sender      = m.Sender()
		isFromAdmin = m.IsFromAdmin
	)

	// Get chat vars
	var (
		cid       = chat.ID
		isPrivate = chat.IsPrivate()
	)

	// Process message based on sender
	var isAllowed bool
	var safeChatQueue *history.SafeChatQueue
	if isFromAdmin { // Message from admin: allowed, new chat queue
		isAllowed = true
	} else { // Ordinary message: validated, shared chat queue
		isAllowed = validateChatID(cid)
		safeChatQueue = shared[cid]
	}

	// Get history by passing nil/shared safe chat queue
	// for admin chats and public chats respectively,
	// nil means new safe chat queue will be created
	SafeChatHistory, _ := sbh.Get(cid, safeChatQueue)

	return &ChatInfo{
		ID:        cid,
		Title:     getChatTitle(chat, sender, isPrivate),
		History:   SafeChatHistory,
		IsAllowed: isAllowed,
		LastMsg:   m,
	}
}

// Gets chat title for any chat
func getChatTitle(
	chat *tg.Chat,
	sender string,
	isPrivate bool,
) string {
	if isPrivate {
		return fmt.Sprintf("%s's private", sender)
	}
	return chat.Title
}
