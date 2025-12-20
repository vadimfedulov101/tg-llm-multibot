package history

import (
	"context"
	"errors"
	"log"
	"runtime"
	"sync"
	"time"

	"tg-handler/conf"
)

// Max procs
var (
	maxProcs = runtime.GOMAXPROCS(0)
)

// Local errors
var (
	ErrSaveFailed = errors.New("[history] failed to save history")
)

// Deletes messages with expired TTL with intervals
func (safeHistory *SafeHistory) Cleaner(
	ctx context.Context,
	historyPath string,
	intervals *conf.CleanerIntervals,
) {
	// Get variables
	var (
		cleanupInterval = time.Duration(intervals.Cleanup)
		messageTTL      = time.Duration(intervals.MessageTTL)
	)

	// Clean up preemptively
	safeHistory.cleanAndSave(historyPath, messageTTL)

	// Start ticker
	t := time.NewTicker(cleanupInterval)
	defer t.Stop()

	// Clean up on tick until queue done
	defer log.Println("Cleaner shut down gracefully.")
	for {
		select {
		case <-t.C:
			safeHistory.cleanAndSave(historyPath, messageTTL)
		case <-ctx.Done():
			log.Println("Cleaner received shutdown signal.")
			return
		}
	}
}

// Cleans in runtime and saves history locally
func (sh *SafeHistory) cleanAndSave(
	historyPath string,
	messageTTL time.Duration,
) {
	// Clean (non-blocking)
	sh.clean(messageTTL)

	// Save history (blocking)
	if err := sh.Save(historyPath); err != nil {
		log.Printf("%v: %v", ErrSaveFailed, err)
	}
}

// Deletes expired messages in safe history
func (sh *SafeHistory) clean(messageTTL time.Duration) {
	// Get bot worker number based on CPU cores
	botWorkerNum := maxProcs / 2

	// Get current time
	currentTime := time.Now()

	// Get all bot names
	sh.mu.RLock()
	botNames := make([]string, 0, len(sh.History))
	for botName := range sh.History {
		botNames = append(botNames, botName)
	}
	sh.mu.RUnlock()

	// Clean bot histories in worker pool
	for i := 0; i < len(botNames); i += botWorkerNum {
		// Calculate batch size
		end := min(i+botWorkerNum, len(botNames))

		// Process batch
		var wg sync.WaitGroup
		for _, botName := range botNames[i:end] {
			wg.Go(func() {
				sh.cleanBot(currentTime, messageTTL, botName)
			})
		}
		wg.Wait()
	}
}

// Deletes expired messages in safe bot history
func (sh *SafeHistory) cleanBot(
	currentTime time.Time,
	messageTTL time.Duration,
	botName string,
) {
	// Get chat worker number based on CPU cores
	chatWorkerNum := maxProcs / 2

	// Get bot data & skip if new
	sbd, ok := sh.Get(botName)
	if !ok {
		return
	}

	// Omit bot contacts (auto-cleaned on reply)
	sbh, _ := sbd.Unpack()

	// Get all chat IDs
	sbh.mu.RLock()
	cids := make([]int64, 0, len(sbh.History))
	for cid := range sbh.History {
		cids = append(cids, cid)
	}
	sbh.mu.RUnlock()

	// Clean chat histories in worker pool
	for i := 0; i < len(cids); i += chatWorkerNum {
		// Calculate batch size
		end := min(i+chatWorkerNum, len(cids))

		// Process batch
		var wg sync.WaitGroup
		for _, cid := range cids[i:end] {
			wg.Go(func() {
				sbh.cleanChat(currentTime, messageTTL, cid)
			})
		}
		wg.Wait()
	}
}

// Deletes expired messages in safe chat history
func (sbh *SafeBotHistory) cleanChat(
	currentTime time.Time,
	messageTTL time.Duration,
	cid int64,
) {
	// Get chat data & skip if new
	sch, ok := sbh.Get(cid)
	if !ok {
		return
	}

	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	// Clean chat queue
	sch.History.ChatQueue.clean(currentTime, messageTTL)

	// Clean reply chain
	sch.History.ReplyChains.clean(currentTime, messageTTL)
}

// Deletes expired messages in chat queue
func (scq *SafeChatQueue) clean(
	currentTime time.Time,
	messageTTL time.Duration,
) {
	// Ensure secure access
	scq.mu.Lock()
	defer scq.mu.Unlock()

	// Get reply chains
	chatQueue := scq.ChatQueue

	// Create new slice pointing to the same array
	queue := chatQueue[:0]

	// Append every not-expired message entry to new slice
	for _, messageEntry := range chatQueue {
		if currentTime.Sub(messageEntry.Timestamp) <= messageTTL {
			queue = append(queue, messageEntry)
		}
	}

	// Set array to new slice
	scq.ChatQueue = queue
}

// Deletes expired messages in reply chains
func (src *SafeReplyChains) clean(
	currentTime time.Time,
	messageTTL time.Duration,
) {
	// Ensure secure access
	src.mu.Lock()
	defer src.mu.Unlock()

	// Get reply chains
	replyChains := src.ReplyChains

	// Delete messages which time of existence
	// is longer than time to live.
	for line, messageEntry := range replyChains {
		if currentTime.Sub(messageEntry.Timestamp) > messageTTL {
			delete(replyChains, line)
		}
	}
}
