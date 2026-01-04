package messaging

import (
	"context"
	"errors"
	"log"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Typing errors
var (
	ErrSignalFailed = errors.New("[messaging] signal request failed")
)

// Sends typing signal until context done
func Type(ctx context.Context, bot *tg.BotAPI, c *ChatInfo) {
	const (
		signal   = "typing"
		interval = 3 * time.Second
	)

	cid := c.ID

	// Type right away
	sendSignal(bot, cid, signal)

	// Set ticker with interval
	t := time.NewTicker(interval)
	defer t.Stop()

	// Type on ticks until context DONE
	for {
		select {
		case <-t.C:
			sendSignal(bot, cid, signal)
		case <-ctx.Done():
			log.Println("[messaging] typing context done")
			return
		}
	}
}

// Sends signal via bot in specific chat
func sendSignal(bot *tg.BotAPI, cid int64, signal string) {
	actConf := tg.NewChatAction(cid, signal)
	_, err := bot.Request(actConf)
	if err != nil {
		log.Printf("%v for <%s>: %v", ErrSignalFailed, signal, err)
	}
}
