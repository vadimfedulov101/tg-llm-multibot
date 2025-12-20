package messaging

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MessageInfo is recursive type.
// We construct chat context and reply chain from it to keep history.
// It implements LineChain (history/memory.go) providing Line() and PrevLine().
type MessageInfo struct {
	Message      *tg.Message
	sender       string       // UserName / FirstName (+LastName)
	line         string       // "Sender: text"
	IsTriggering bool         // Triggering messages get replied
	IsVIP        bool         // VIP allows bypass checks
	prevMsg      *MessageInfo // Previous message info
}

// MessageInfo constructor relies on bot dictating how to detect admin, reply,
// mentions; modify mentions. This dependency inversion enables to get suitable
// Text and IsTriggering/IsVIP as direct message traits needed for validation.
func NewMessageInfo(
	bot *tg.BotAPI,
	msg *tg.Message,
	validateSender func(*tg.Message, string) bool,
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

	// Check if sender is admin (chat is allowed for their username)
	isFromAdmin := validateSender(msg, sender)

	// Check if replied
	isReplied := detectReply(msg)

	// Check if mentioned; modify mentions
	isMentioned := detectMentions(text)
	if isMentioned {
		text = modifyMentions(text)
	}

	return &MessageInfo{
		Message:      msg,
		sender:       sender,
		line:         getLine(sender, text),
		IsTriggering: isFromAdmin || isReplied || isMentioned,
		IsVIP:        isFromAdmin,
		prevMsg: NewMessageInfo(
			bot, msg.ReplyToMessage,
			validateSender,
			detectReply,
			detectMentions,
			modifyMentions,
			level+1,
		),
	}
}

// Line exposed (history.LineChain & model.Message implemented)
func (m *MessageInfo) Line() string {
	return m.line
}

// Previous line exposed (history.LineChain implemented)
func (m *MessageInfo) PrevLine() string {
	prevMsg := m.prevMsg
	if prevMsg != nil {
		return prevMsg.Line()
	}
	return ""
}

// Sender exposed (model.Message implemented)
func (m *MessageInfo) Sender() string {
	return m.sender
}

// Gets any name from UserName/FirstName (+LastName)
func getSender(msg *tg.Message) string {
	return msg.From.String()
}

// Gets any text from Text/Caption
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
