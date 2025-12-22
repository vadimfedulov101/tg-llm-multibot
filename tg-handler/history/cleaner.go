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

// Cleaner errors
var (
	ErrSaveFailed = errors.New("[cleaner] failed to save history")
)

// Context message variations
const (
	// Main message
	ctxDoneMsg = "[cleaner] received shutdown signal upon"

	// Operations
	opCleanAndSave = "clean and save operation"
	opSendingJobs  = "sending jobs to channel"
	opGettingJobs  = "getting jobs from channel"
)

// Single unit of update
type CleanJob struct {
	BotName string
	CID     int64
}

// Deletes messages with expired TTL every clean up interval
func (safeHistory *SafeHistory) Cleaner(
	ctx context.Context,
	path string,
	settings *conf.CleanerSettings,
) {
	// Get variables
	var (
		cleanupInterval = time.Duration(settings.CleanupInterval)
		messageTTL      = time.Duration(settings.MessageTTL)
	)

	// Start ticker
	t := time.NewTicker(cleanupInterval)
	defer t.Stop()

	// Clean and save on tick until context done
	defer log.Println("Cleaner shut down gracefully")
	for {
		select {
		case <-t.C:
			done := safeHistory.cleanAndSave(ctx, path, messageTTL)
			if done { // Check deeper context done
				return
			}
		case <-ctx.Done():
			log.Println(ctxDoneMsg + opCleanAndSave)
			return
		}
	}
}

// Cleans in runtime and saves history locally
func (sh *SafeHistory) cleanAndSave(
	ctx context.Context,
	historyPath string,
	messageTTL time.Duration,
) bool {
	// Clean
	done := sh.clean(ctx, messageTTL)
	if done { // Check deeper context done
		return true
	}

	// Save history
	if err := sh.Save(historyPath); err != nil {
		log.Printf("%v: %v", ErrSaveFailed, err)
	}

	return false
}

// Deletes expired messages in safe history
func (sh *SafeHistory) clean(
	ctx context.Context,
	messageTTL time.Duration,
) bool {
	var wg sync.WaitGroup

	// Get current time
	currentTime := time.Now()

	// COLLECT all jobs
	jobs := sh.collectCleanJobs()
	if len(jobs) == 0 {
		return false
	}

	// CREATE worker pool (no more workers than jobs)
	workerCount := min(runtime.GOMAXPROCS(0), len(jobs))
	jobsChan := make(chan CleanJob, len(jobs))
	doneChan := make(chan struct{})

	// START workers
	go func() {
		for workerID := range workerCount {
			wg.Go(func() {
				sh.cleanWorker(
					ctx, currentTime, messageTTL,
					jobsChan, workerID, doneChan,
				)
			})
		}
	}()

	// SEND jobs to channel until context DONE
	go func() {
		defer close(jobsChan) // All jobs sent

		for _, job := range jobs {
			select {
			case jobsChan <- job:
			case <-doneChan:
				return
			case <-ctx.Done():
				log.Println(ctxDoneMsg + opSendingJobs)
				doneChan <- struct{}{}
			}
		}
	}()

	// WAIT for completion
	wg.Wait()

	return false
}

func (sh *SafeHistory) collectCleanJobs() []CleanJob {
	var jobs []CleanJob

	sh.mu.RLock()
	defer sh.mu.RUnlock()

	// Iterate over safe bot data
	for botName, sbd := range sh.History {
		sbd.mu.RLock()

		// Get only safe bot history
		sbh := &sbd.Data.History // Safe bot history is auto-cleaned

		// Iterate over safe bot history
		sbh.mu.RLock()
		for cid := range sbh.History {
			jobs = append(jobs, CleanJob{
				BotName: botName,
				CID:     cid,
			})
		}
		sbh.mu.RUnlock()

		sbd.mu.RUnlock()
	}

	log.Printf("[cleaner] collected %d jobs", len(jobs))
	return jobs
}

// Processess clean jobs with context
func (sh *SafeHistory) cleanWorker(
	ctx context.Context,
	currentTime time.Time,
	messageTTL time.Duration,
	jobsChan <-chan CleanJob,
	workerID int,
	doneChan chan<- struct{},
) {
	// GET jobs from channel jobs channel CLOSED or context DONE
	for processed := 0; ; processed++ {
		select {
		case job, ok := <-jobsChan:
			if !ok { // Channel closed
				log.Println("[cleaner] jobs channel closed")
				log.Printf(
					"[cleaner] worker %d processed %d jobs",
					workerID, processed,
				)
				return
			}
			sh.processCleanJob(currentTime, messageTTL, job)
		case <-ctx.Done():
			log.Println(ctxDoneMsg + opGettingJobs)
			doneChan <- struct{}{}
			return
		}
	}
}

// Handles single chat cleanup
func (sh *SafeHistory) processCleanJob(
	currentTime time.Time,
	messageTTL time.Duration,
	job CleanJob,
) {
	// Get chat directly without creating new ones
	sch, ok := sh.GetChatHistory(job.BotName, job.CID)
	if !ok {
		return
	}

	// Clean both structures
	sch.History.ChatQueue.clean(currentTime, messageTTL)
	sch.History.ReplyChains.clean(currentTime, messageTTL)
}

func (sh *SafeHistory) getChatHistory(
	botName string,
	chatID int64,
) (*SafeChatHistory, bool) {
	// Get safe bot data
	sbd, ok := sh.Get(botName)
	if !ok {
		return nil, false
	}

	// Get only safe bot history
	sbh, _ := sbd.Get() // Omit safe bot contacts as auto-cleaned

	// Get safe chat history
	sch, ok := sbh.Get(chatID)
	return sch, ok
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
