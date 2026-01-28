package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"tg-handler/bot"
	"tg-handler/conf"
	"tg-handler/history"
	"tg-handler/logging"
	"tg-handler/secret"
)

const InitConfPath = "./confs/init.json"

func main() {
	// Get logger
	logger := logging.New(slog.LevelInfo)

	// Load API keys from secret file or panic
	apiKeys := secret.MustLoadAPIKeys(logger)

	// Terminate on termination signal
	ctx, cancel := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer cancel()

	// Get init config
	iConf := conf.MustLoadInitConf(InitConfPath, logger)

	// Get safe history
	history := history.MustLoadHistory(
		iConf.Paths.History,
		// Preinitialize SafeChatQueues with allowed chat IDs
		iConf.BotSettings.AllowedChats.IDs,
		logger,
	)

	// Start cleaner and bots
	wg, updateCh := startBots(ctx, iConf, apiKeys, history, logger)

	// Await termination signal
	<-ctx.Done()

	// Shut down
	logger.Info("shutting down...")
	logger.Info("awaiting services to shutdown...")
	wg.Wait()
	close(updateCh)
	logger.Info("all services shutdown gracefully")
}

// Starts bots with API keys
func startBots(
	ctx context.Context,
	iConf *conf.InitConf,
	apiKeys []string,
	history *history.History,
	logger *logging.Logger,
) (*sync.WaitGroup, chan any) {
	var (
		wg       sync.WaitGroup
		updateCh = make(chan any)

		historyPath = iConf.Paths.History
	)

	// Start cleaner
	wg.Go(func() {
		history.Cleaner(
			ctx, historyPath, &iConf.CleanerSettings, logger,
		)
	})

	// Start all bots
	for _, apiKey := range apiKeys {
		wg.Go(func() {
			bot := bot.New(
				apiKey, iConf, history, updateCh, &wg, logger,
			)
			bot.Start(ctx)
		})
	}

	// Start history saver
	wg.Go(func() {
		history.Saver(ctx, historyPath, updateCh, logger)
	})

	return &wg, updateCh
}
