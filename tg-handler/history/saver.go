package history

import (
	"context"
	"log"
)

// Saves safe history on each update signal
func (safeHistory *SafeHistory) Saver(
	ctx context.Context, historyPath string, updateSignalCh <-chan any,
) {
	// Save on update signal until context done
	defer log.Println("Saver shut down gracefully.")
	for {
		select {
		case _, ok := <-updateSignalCh:
			if !ok {
				log.Println("History update channel was closed.")
				return
			}
			safeHistory.Save(historyPath)
		case <-ctx.Done():
			log.Println("Saver received shutdown signal.")
			return
		}
	}
}
