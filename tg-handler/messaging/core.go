package messaging

import (
	"fmt"
	"slices"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Message information is a recursive type.
type MessageInfo struct {
	Message *tg.Message
	Sender  string
	Text    string
	Line    string

	Prev *MessageInfo
}

// Creates message info.
// We store message info in history to transform into memory,
// so we don't rely on Telegram providing us more than one previous message.
func NewMessageInfo(bot *tg.BotAPI, msg *tg.Message, level int) *MessageInfo {
	// Handle nil and high recursion
	if msg == nil || level > 2 {
		return nil
	}

	// Get sender and text
	var (
		sender = getSender(msg)
		text   = getText(msg)
	)

	// Humanize bot mention in user written text
	if !msg.From.IsBot {
		text = humanizeBotMention(bot, text)
	}

	return &MessageInfo{
		Message: msg,
		Sender:  sender,
		Text:    text,
		Line:    getLine(sender, text),
		Prev:    NewMessageInfo(bot, msg.ReplyToMessage, level+1),
	}
}

// Chat information extends message info
type ChatInfo struct {
	MessageInfo
	CID       int64
	ChatTitle string
}

func NewChatInfo(m *MessageInfo) *ChatInfo {
	var (
		msg    = m.Message
		sender = m.Sender
	)

	return &ChatInfo{
		MessageInfo: *m,
		CID:         getCID(msg),
		ChatTitle:   getChatTitle(msg, sender),
	}
}

// Checks if bot is asked
func (c *ChatInfo) IsAsked(bot *tg.BotAPI, admins []string, cids []int64) bool {
	// Get vars
	var (
		msg    = c.Message
		sender = c.Sender
		text   = c.Text
	)

	// Reply privately only to admins
	if msg.Chat.IsPrivate() { // Username check
		return slices.Contains(admins, sender)
	}

	// Reply publicly only in allowed chats
	allowed := false
	for _, cid := range cids { // CID check
		if cid == msg.Chat.ID {
			allowed = true
		}
	}
	if !allowed {
		return false
	}

	// Get replied message if any
	replied := msg.ReplyToMessage
	var repliedID int64
	if replied != nil {
		repliedID = replied.From.ID
	}

	// Conditions
	var (
		isReplied   = repliedID == bot.Self.ID
		isMentioned = strings.Contains(text, bot.Self.FirstName)
		isCommanded = msg.IsCommand()
	)

	return isReplied || isMentioned || isCommanded
}

// Gets sender for bots and users
func getSender(msg *tg.Message) (sender string) {
	// Set first name for bot (human-like informal naming)
	if msg.From.IsBot {
		return msg.From.FirstName
	}

	// Set username for user (unique and concise naming)
	sender = msg.From.UserName
	if sender == "" { // Or fall back to first name
		sender = msg.From.FirstName
	}

	return sender
}

// Gets any text from message
func getText(msg *tg.Message) (text string) {
	if msg.Text != "" {
		text = msg.Text
	}
	if msg.Caption != "" {
		text = msg.Caption
	}

	return text
}

// Gets line in format "Sender: text"
func getLine(sender string, text string) string {
	titleizer := cases.Title(language.English)
	return titleizer.String(sender) + ": " + text
}

// Substitutes bot's @username to FirstName in text
func humanizeBotMention(bot *tg.BotAPI, text string) string {
	text = strings.ReplaceAll(
		text, "@"+bot.Self.UserName, bot.Self.FirstName+",",
	)
	return text
}

// Gets chat ID for public and private chats
func getCID(msg *tg.Message) (cid int64) {
	if msg.Chat != nil {
		cid = msg.Chat.ID
	} else {
		cid = msg.From.ID
	}

	return cid
}

// Gets chat title for public and private chats
func getChatTitle(msg *tg.Message, sender string) (chatTitle string) {
	chatTitle = msg.Chat.Title
	if chatTitle == "" {
		chatTitle = fmt.Sprintf("%s's chat", sender)
	}

	return chatTitle
}
