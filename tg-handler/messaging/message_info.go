package messaging

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Recursive type.
// Provides Line(), PrevLine() methods to construct reply chain.
// Provides ID() and Sender() as methods.
type MessageInfo struct {
	ID           int    // Message identifier
	sender       string // UserName | FirstName (+LastName)
	line         string // "Sender: text"
	IsTriggering bool   // Is message meant to be replied
	IsFromAdmin  bool   // Is message meant to be queued privately
	Chat         *tg.Chat
	prevMsg      *MessageInfo // Previous message info
}

// Constructs message info by following bot procedures
// on how to detect admin/reply/mentions; modify mentions.
func NewMessageInfo(
	bot *tg.BotAPI,
	msg *tg.Message,
	detectAdmin func(*tg.Message, string) bool,
	detectReply func(*tg.Message) bool,
	detectMentions func(string) bool,
	modifyMentions func(string) string,
	level int,
) *MessageInfo {
	// Handle nil and too deep recursion
	if msg == nil || level > 2 {
		return nil
	}

	// Get sender and text
	var (
		sender = getSender(msg)
		text   = getText(msg)
	)
	// Return nil if no sender or text
	if sender == "" || text == "" {
		return nil
	}

	// Get basic info
	var (
		isFromAdmin = detectAdmin(msg, sender)
		isReplied   = detectReply(msg)
		isMentioned = detectMentions(text)
	)

	// Modify bot mentions if they exist
	if isMentioned {
		text = modifyMentions(text)
	}

	return &MessageInfo{
		Chat:         msg.Chat,
		ID:           msg.MessageID,
		sender:       sender,
		line:         getLine(sender, text),
		IsTriggering: isFromAdmin || isReplied || isMentioned,
		IsFromAdmin:  isFromAdmin,
		prevMsg: NewMessageInfo(
			bot, msg.ReplyToMessage,
			detectAdmin,
			detectReply,
			detectMentions,
			modifyMentions,
			level+1,
		),
	}
}

// Line exposed
func (m *MessageInfo) Line() string {
	return m.line
}

// Previous line exposed
func (m *MessageInfo) PrevLine() string {
	prevMsg := m.prevMsg
	if prevMsg != nil {
		return prevMsg.Line()
	}
	return ""
}

// Sender exposed
func (m *MessageInfo) Sender() string {
	return m.sender
}

// Gets UserName | FirstName (+LastName)
func getSender(msg *tg.Message) string {
	return msg.From.String()
}

// Gets Text | Caption
func getText(msg *tg.Message) (text string) {
	if msg.Text != "" {
		text = msg.Text
	}
	if msg.Caption != "" {
		text = msg.Caption
	}
	return text
}

// Gets "Sender: text" message history representation
func getLine(sender string, text string) string {
	titleizer := cases.Title(language.English)
	return titleizer.String(sender) + ": " + text
}
