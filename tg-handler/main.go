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
	h := history.MustLoadHistory(iConf.Paths.History)
	sh := history.NewSafeHistory(h)

	// Start cleaner and bots
	wg, updateCh := startBots(ctx, iConf, apiKeys, sh)

	// Await termination signal
	<-ctx.Done()
	log.Println("Shutting down...")

	// Await bots shutdown
	log.Println("Waiting for bot services to shutdown...")
	wg.Wait()
	close(updateCh)
	log.Println("All bot services shutdown gracefully")

	// Save history (maybe no need in final save?)
	// log.Println("Saving history...")
	// sh.Save(iConf.Paths.History)
	// log.Println("History saved")
}

// Starts bots with API keys
func startBots(
	ctx context.Context,
	iConf *conf.InitConf,
	apiKeys []string,
	sh *history.SafeHistory,
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
		sh.Cleaner(ctx, historyPath, &iConf.CleanerSettings)
	}()

	// Start all bots
	for _, apiKey := range apiKeys {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bot := bot.New(apiKey, iConf, sh, updateCh)
			bot.Start(ctx)
		}()
	}

	// Start history saver
	wg.Add(1)
	go func() {
		defer wg.Done()
		sh.Saver(ctx, historyPath, updateCh)
	}()

	return &wg, updateCh
}
