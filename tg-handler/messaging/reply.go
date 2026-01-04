package messaging

import (
	"errors"
	"fmt"
	"log"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Messaging errors
var (
	ErrDirectReplyFailed   = errors.New("[messaging] direct reply failed")
	ErrIndirectReplyFailed = errors.New("[messaging] indirect reply failed")
)

// Template for formatting replied line and reply text on second try
const ReplyDeletedT = "> '%s'\n\n%s"

// Try to reply twice: with reply, with separate message
func Reply(bot *tg.BotAPI, c *ChatInfo, text string) *tg.Message {
	var (
		mid = c.LastMsg.ID
		cid = c.ID
	)

	// Get and set message config
	m := tg.NewMessage(cid, text)
	m.ReplyToMessageID = mid

	// Try to reply with reply
	response, err := bot.Send(m)
	if err != nil { // Try to reply with separate message
		log.Printf("%v: %v", ErrDirectReplyFailed, err)
		m.ReplyToMessageID = 0
		m.Text = fmt.Sprintf(ReplyDeletedT, c.LastMsg.Line(), text)
		response, err = bot.Send(m)
	}
	if err != nil {
		log.Printf("%v: %v", ErrIndirectReplyFailed, err)
	}

	return &response
}
