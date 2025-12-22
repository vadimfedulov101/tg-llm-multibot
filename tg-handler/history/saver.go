package history

import (
	"context"
	"log"
)

// Saves safe history on each update signal
func (safeHistory *SafeHistory) Saver(
	ctx context.Context, path string, updateCh <-chan any,
) {
	// Save history on signal until channel CLOSED or context DONE
	defer log.Println("[history] saver shut down gracefully")
	for {
		select {
		case _, ok := <-updateCh:
			if !ok { // Check if update channel closed
				log.Println("[history] update channel was closed")
				return
			}
			safeHistory.Save(path)
		case <-ctx.Done():
			log.Println("[history] saver received shutdown signal")
			return
		}
	}
}
