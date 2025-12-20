package messaging

import (
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/history"
)

// ChatInfo stores chat's ID, title, history, status and last message
type ChatInfo struct {
	ID        int64
	Title     string
	History   *history.SafeChatHistory
	IsAllowed bool
	LastMsg   *MessageInfo
}

func NewChatInfo(
	m *MessageInfo,
	h *history.SafeBotHistory,
	validateChat func(*tg.Message, int64) bool,
) *ChatInfo {
	// Get message and sender
	var (
		msg    = m.Message
		sender = m.Sender()
		isVIP  = m.IsVIP
	)

	// Get chat
	chat := msg.Chat

	// Get chat ID and type
	var (
		cid       = chat.ID
		isPrivate = chat.IsPrivate()
	)

	// Check if chat is allowed
	var isAllowed bool
	if isVIP {
		isAllowed = true
	} else {
		isAllowed = validateChat(msg, cid)
	}

	return &ChatInfo{
		ID:        cid,
		Title:     getChatTitle(msg, sender, isPrivate),
		History:   h.Get(cid),
		IsAllowed: isAllowed,
		LastMsg:   m,
	}
}

// Gets chat title for public and private chats
func getChatTitle(msg *tg.Message, sender string, isPrivate bool) string {
	if isPrivate {
		return fmt.Sprintf("%s's private", sender)
	}
	return msg.Chat.Title
}
