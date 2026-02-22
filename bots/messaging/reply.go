package messaging

import (
	"errors"
	"fmt"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"telellama/logging"
)

// Messaging errors
var (
	errDirectReplyFailed   = errors.New("direct reply failed")
	errIndirectReplyFailed = errors.New("indirect reply failed")
)

const (
	maxRetries = 10
	retryDelay = 5 * time.Second
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
	response, err := sendWithRetry(bot, m)
	if err != nil { // Try to reply with separate message
		logger.Error(errDirectMsg, logging.Err(
			fmt.Errorf("%w: %v", errDirectReplyFailed, err),
		))

		m.ReplyToMessageID = 0
		m.Text = fmt.Sprintf(ReplyToDelT, c.LastMsg.Line(), text)
		response, err = sendWithRetry(bot, m)
	}
	if err != nil {
		logger.Error(errIndirectMsg, logging.Err(
			fmt.Errorf("%w: %v", errIndirectReplyFailed, err),
		))
	}

	return &response
}

// Helper to retry sending messages on temporary network failures
func sendWithRetry(bot *tg.BotAPI, msg tg.MessageConfig) (tg.Message, error) {
	var err error
	var resp tg.Message

	for i := range maxRetries {
		resp, err = bot.Send(msg)
		if err == nil {
			return resp, nil
		}

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}
	return resp, err
}
