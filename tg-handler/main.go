package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"tg-handler/bot"
	"tg-handler/initconf"
	"tg-handler/memory"
)

const InitConfPath = "./confs/init.json"

func main() {
	// Terminate on termination signal gracefully
	ctx, cancel := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer cancel()

	// Load initialization config
	initConf, err := initconf.Load(InitConfPath)
	if err != nil {
		log.Fatalf("Failed to load init config: %v", err)
	}

	// Get safe history and cleaner running
	HistoryPath := initConf.HistoryPath
	h, err := memory.LoadHistory(HistoryPath)
	if err != nil {
		log.Fatalf("Failed to load history: %v", err)
	}
	sh := memory.NewSafeHistory(h)
	go memory.Cleaner(ctx, sh, HistoryPath, &initConf.MemoryConfig)

	// Start all bots with shared history
	var wg sync.WaitGroup
	for id := range initConf.KeysAPI {
		wg.Add(1)
		go func(botId int) {
			defer wg.Done()
			bot.StartWithCtx(ctx, botId, initConf, sh)
		}(id)
	}

	// Block until termination signal
	<-ctx.Done()
	log.Println("Shutting down...")

	log.Println("Waiting for bots to shutdown...")
	wg.Wait()
	log.Println("All bots shutdown gracefully")
}
