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
)

const InitConfPath = "./confs/init.json"

func main() {
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
	wg, historyUpdSignalCh := startBots(ctx, iConf, sh)

	// Await termination signal
	<-ctx.Done()
	log.Println("Shutting down...")

	// Await bots shutdown
	log.Println("Waiting for bot services to shutdown...")
	wg.Wait()
	close(historyUpdSignalCh)
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
	sh *history.SafeHistory,
) (*sync.WaitGroup, chan any) {
	var (
		wg                 sync.WaitGroup
		historyUpdSignalCh = make(chan any)

		historyPath = iConf.Paths.History
	)

	// Start cleaner
	wg.Add(1)
	go func() {
		defer wg.Done()
		sh.Cleaner(ctx, historyPath, &iConf.CleanerIntervals)
	}()

	// Start all bots
	for _, keyAPI := range iConf.KeysAPI {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bot := bot.New(keyAPI, iConf, sh, historyUpdSignalCh)
			bot.Start(ctx)
		}()
	}

	// Start history saver
	wg.Add(1)
	go func() {
		defer wg.Done()
		sh.Saver(ctx, historyPath, historyUpdSignalCh)
	}()

	return &wg, historyUpdSignalCh
}
