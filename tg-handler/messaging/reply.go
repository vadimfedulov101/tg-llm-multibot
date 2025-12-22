package messaging

import (
	"errors"
	"log"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Messaging errors
var (
	ErrDirectReplyFailed   = errors.New("[messaging] direct reply failed")
	ErrIndirectReplyFailed = errors.New("[messaging] indirect reply failed")
)

// Try to reply twice: with reply, with separate message
func Reply(bot *tg.BotAPI, c *ChatInfo, text string) *tg.Message {
	var (
		msg    = c.LastMsg.Message
		chatID = c.ID
	)

	// Construct message config and set it up for reply
	m := tg.NewMessage(chatID, text)
	m.ReplyToMessageID = msg.MessageID

	// Try to reply with reply
	response, err := bot.Send(m)
	if err != nil { // Try to reply with separate message
		log.Printf("%v: %v", ErrDirectReplyFailed, err)
		m.ReplyToMessageID = 0
		response, err = bot.Send(m)
	}
	if err != nil {
		log.Printf("%v: %v", ErrIndirectReplyFailed, err)
	}

	return &response
}
