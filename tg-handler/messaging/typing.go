package messaging

import (
	"context"
	"errors"
	"fmt"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/logging"
)

// Typing errors
var (
	errSignalFailed = errors.New("signal request failed")
)

// Sends typing signal until context done
func Type(
	ctx context.Context, bot *tg.BotAPI, c *ChatInfo,
	logger *logging.Logger,
) {
	// Set constants
	const (
		signal   = "typing"
		interval = 3 * time.Second
	)
	logger = logger.With(logging.Signal(signal))

	cid := c.ID

	// Type right away
	sendSignal(bot, cid, signal, logger)

	// Set ticker with interval
	t := time.NewTicker(interval)
	defer t.Stop()

	// Type on ticks until context DONE
	for {
		select {
		case <-t.C:
			sendSignal(bot, cid, signal, logger)
		case <-ctx.Done():
			logger.Debug("typing context done")
			return
		}
	}
}

// Sends signal via bot in specific chat
func sendSignal(
	bot *tg.BotAPI, cid int64, signal string,
	logger *logging.Logger,
) {
	// Set error message
	const errMsg = "signal send failed"

	// Try to send signal
	actConf := tg.NewChatAction(cid, signal)
	_, err := bot.Request(actConf)
	if err != nil {
		logger.Error(errMsg, logging.Err(
			fmt.Errorf("%w: %v", errSignalFailed, err)),
		)
	}
}
