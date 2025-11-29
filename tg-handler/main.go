package main

import (
	"context"
	"log"
	"os"
	"os/signal"
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
	h := history.MustLoadHistory(iConf.PathsConf.History)
	sh := history.NewSafeHistory(h)

	// Start cleaner and bots
	go history.Cleaner(ctx, iConf.PathsConf.History, sh, &iConf.CleanerConf)
	wg := bot.StartBots(ctx, iConf, sh)

	// Await termination signal
	<-ctx.Done()
	log.Println("Shutting down...")

	// Await bots shutdown
	log.Println("Waiting for bots to shutdown...")
	wg.Wait()
	log.Println("All bots shutdown gracefully")

	// Save history
	log.Println("Saving history...")
	sh.Save(iConf.PathsConf.History)
	log.Println("History saved")
}
