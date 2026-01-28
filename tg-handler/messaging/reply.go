package messaging

import (
	"errors"
	"fmt"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/logging"
)

// Messaging errors
var (
	errDirectReplyFailed   = errors.New("direct reply failed")
	errIndirectReplyFailed = errors.New("indirect reply failed")
)

// Try to reply twice: with reply, with separate message
func Reply(
	bot *tg.BotAPI, c *ChatInfo, text string,
	logger *logging.Logger,
) *tg.Message {
	// Set template (for replying already deleted messages)
	const ReplyToDelT = "> '%s'\n\n%s"
	// Set error messages
	const (
		errDirectMsg   = "direct reply failed"
		errIndirectMsg = "indirect reply failed"
	)

	var (
		msgID  = c.LastMsg.ID
		chatID = c.ID
	)

	// Get and set message config
	m := tg.NewMessage(chatID, text)
	m.ReplyToMessageID = msgID

	// Try to reply with reply
	response, err := bot.Send(m)
	if err != nil { // Try to reply with separate message
		logger.Error(errDirectMsg, logging.Err(
			fmt.Errorf("%w: %v", errDirectReplyFailed, err),
		))

		m.ReplyToMessageID = 0
		m.Text = fmt.Sprintf(ReplyToDelT, c.LastMsg.Line(), text)
		response, err = bot.Send(m)
	}
	if err != nil {
		logger.Error(errIndirectMsg, logging.Err(
			fmt.Errorf("%w: %v", errIndirectReplyFailed, err),
		))
	}

	return &response
}
