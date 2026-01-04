package history

import (
	"context"
	"errors"
	"log"
)

// Saver errors
var (
	ErrFinalSaveFailed = errors.New("[history] final save failed")
)

// Saves history on update signal
func (history *History) Saver(
	ctx context.Context, path string, updateCh <-chan any,
) {
	// Save history on signal until channel CLOSED or context DONE
	defer log.Println("[history] saver shut down gracefully")
	for {
		select {
		case _, ok := <-updateCh:
			if !ok { // Check if channel closed
				log.Println("[history] update channel was closed")
				return
			}
			history.Save(path)
		case <-ctx.Done():
			log.Println("[history] saver received shutdown signal")
			if err := history.Save(path); err != nil {
				log.Printf("%v: %v", ErrFinalSaveFailed, err)
			}
			return
		}
	}
}
