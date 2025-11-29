package bot

import (
	"context"
	"errors"
	"log"
	"sync"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/conf"
	"tg-handler/history"
)

// Bot errors
var (
	ErrAuth = errors.New("[bot] authorization failed")
)

// Bot constants
const (
	timeout = 30
)

// Starts bots with API keys
func StartBots(
	ctx context.Context,
	iConf *conf.InitConf,
	safeHistory *history.SafeHistory,
) *sync.WaitGroup {
	var wg sync.WaitGroup

	for _, keyAPI := range iConf.BotsConf.KeysAPI {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startBot(ctx, keyAPI, iConf, safeHistory)
		}()
	}

	return &wg
}

// Starts bot
func startBot(
	ctx context.Context,
	keyAPI string,
	iConf *conf.InitConf,
	safeHistory *history.SafeHistory,
) {
	// Authorize as bot
	bot, err := tg.NewBotAPI(keyAPI)
	if err != nil {
		log.Panicf("%v: %v", ErrAuth, err)
	}
	botName := bot.Self.UserName
	log.Printf("Authorized as %s", botName)

	// Prepare updates channel
	u := tg.NewUpdate(0)
	u.Timeout = timeout
	updates := bot.GetUpdatesChan(u)

	// Handle updates until context done
	defer log.Printf("Bot %s shut down gracefully", botName)
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				log.Printf("Bot %s update channel closed", botName)
				return
			}
			go handleUpdate(ctx, update, iConf, safeHistory, bot, botName)
		case <-ctx.Done():
			log.Printf("Bot %s received shutdown signal", botName)
			return
		}
	}
}
