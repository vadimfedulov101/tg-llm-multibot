package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"tg-handler/bot"
	"tg-handler/conf"
	"tg-handler/history"
	"tg-handler/secret"
)

const InitConfPath = "./confs/init.json"

func main() {
	// Load API keys from secret file or panic
	apiKeys := secret.MustLoadAPIKeys()

	// Terminate on termination signal
	ctx, cancel := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer cancel()

	// Get init config
	iConf := conf.MustLoadInitConf(InitConfPath)

	// Get safe history
	h := history.MustLoadHistory(
		iConf.Paths.History,
		// Preinitialize SafeChatQueues with allowed chat IDs
		iConf.BotSettings.AllowedChats.IDs,
	)

	// Start cleaner and bots
	wg, updateCh := startBots(ctx, iConf, apiKeys, h)

	// Await termination signal
	<-ctx.Done()
	log.Println("[main] shutting down...")

	// Await bots shutdown
	log.Println("[main] awaiting services to shutdown...")
	wg.Wait()
	close(updateCh)
	log.Println("[main] all services shutdown gracefully")
}

// Starts bots with API keys
func startBots(
	ctx context.Context,
	iConf *conf.InitConf,
	apiKeys []string,
	h *history.History,
) (*sync.WaitGroup, chan any) {
	var (
		wg       sync.WaitGroup
		updateCh = make(chan any)

		historyPath = iConf.Paths.History
	)

	// Start cleaner
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.Cleaner(ctx, historyPath, &iConf.CleanerSettings)
	}()

	// Start all bots
	for _, apiKey := range apiKeys {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bot := bot.New(apiKey, iConf, h, updateCh)
			bot.Start(ctx)
		}()
	}

	// Start history saver
	wg.Add(1)
	go func() {
		defer wg.Done()
		h.Saver(ctx, historyPath, updateCh)
	}()

	return &wg, updateCh
}
