package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"tg-handler/bot"
	"tg-handler/initconf"
	"tg-handler/memory"
)

const (
	InitConf        = "./confs/init.json"
	MessageTTL      = 24 * time.Hour
	CleanupInterval = time.Hour
)

func main() {
	// Terminate on termination signal gracefully
	ctx, cancel := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer cancel()

	// Load initialization config
	initJSON := initconf.Load(InitConf)

	// Get KeysAPI and HistoryPath
	KeysAPI := initJSON.KeysAPI
	HistoryPath := initJSON.HistoryPath

	// Make safe history
	h := memory.LoadHistory(HistoryPath)
	sh := memory.NewSafeHistory(h)

	// Start cleaner for common history
	go bot.Cleaner(ctx, sh, HistoryPath, CleanupInterval, MessageTTL)

	// Start all bots with shared history and mutex
	var wg sync.WaitGroup
	for id := range KeysAPI {
		wg.Add(1)
		go func(botId int) {
			defer wg.Done()
			bot.StartWithCtx(ctx, botId, initJSON, sh)
		}(id)
	}

	// Block until termination signal
	<-ctx.Done()
	log.Println("Shutting down...")

	log.Println("Waiting for bots to shutdown...")
	wg.Wait()
	log.Println("All bots shutdown gracefully")
}
