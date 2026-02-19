package history

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"tg-handler/conf"
	"tg-handler/logging"
)

// Cleaner errors
var (
	errSaveFailed = errors.New("failed to save history")
)

// Cleaner context logs
const (
	// Event message
	ctxDoneMsg = "cleaner received shutdown signal upon "

	// Interrupted operations
	opWaiting     = "waiting"
	opSendingJobs = "sending jobs to channel"
	opGettingJobs = "getting jobs from channel"
)

// Single clean unit
type CleanJob struct {
	ChatQueue   *SafeChatQueue
	ReplyChains *SafeReplyChains
}

// Performs clean job
func (j *CleanJob) perform(
	currentTime time.Time,
	messageTTL time.Duration,
) {
	var (
		chatQueue   = j.ChatQueue
		replyChains = j.ReplyChains
	)

	// Perform conditional cleaning
	if chatQueue != nil {
		chatQueue.clean(currentTime, messageTTL)
	}
	if replyChains != nil {
		replyChains.clean(currentTime, messageTTL)
	}
}

// Deletes expired messages with interval and saves cleaned history
func (h *History) Cleaner(
	ctx context.Context,
	path string,
	settings *conf.CleanerSettings,
	logger *logging.Logger,
) {
	// Get variables
	var (
		cleanupInterval = time.Duration(settings.CleanupInterval)
		messageTTL      = time.Duration(settings.MessageTTL)
	)

	// Start ticker
	t := time.NewTicker(cleanupInterval)
	defer t.Stop()

	// Clean and save on tick until context DONE
	defer logger.Info("cleaner shut down gracefully")
	for {
		select {
		case <-t.C:
			// Clean
			err := h.clean(ctx, messageTTL, logger)
			if err != nil && errors.Is(err, context.Canceled) {
				return
			}
			// Save (skip extra context check)
			h.Save(path, logger)
		case <-ctx.Done():
			logger.Info(ctxDoneMsg + opWaiting)
			return
		}
	}
}

// Deletes expired messages in history
func (h *History) clean(
	ctx context.Context,
	messageTTL time.Duration,
	logger *logging.Logger,
) error {
	var currentTime = time.Now()

	// CREATE error group & context
	// When ctx done, gctx done for all workers with logging once
	g, gctx := errgroup.WithContext(ctx)
	var logSendingOnce, logGettingOnce sync.Once

	// COLLECT jobs
	jobs := h.collectCleanJobs(logger)
	if len(jobs) < 1 {
		return nil
	}

	// START job sender
	jobsChan := make(chan CleanJob, len(jobs))
	g.Go(func() error { // evaluates to error
		return jobSender(
			gctx, jobs, jobsChan, &logSendingOnce, logger,
		)
	})

	// START job receivers (clean workers)
	workerCount := min(runtime.GOMAXPROCS(0), len(jobs))
	for workerID := range workerCount {
		g.Go(func() error {
			return h.cleanWorker( // evaluates to error
				gctx, workerID, jobsChan, currentTime, messageTTL,
				&logGettingOnce, logger,
			)
		})
	}

	return g.Wait() // evaluates to error
}

func (h *History) collectCleanJobs(logger *logging.Logger) []CleanJob {
	var jobs []CleanJob

	var (
		scqs = h.SharedChatQueues
		bots = h.Bots
	)

	// Add SHARED chat queues to jobs
	for _, scq := range scqs {
		jobs = append(jobs, CleanJob{
			ChatQueue: scq,
		})
	}

	// Add LOCAL chat queues & reply chains to jobs
	bots.mu.RLock()
	defer bots.mu.RUnlock()
	// Iterate over bot data
	for _, botData := range bots.History {
		sbh := botData.History // Omit contacts

		sbh.mu.RLock()
		// Iterate over chat histories
		for _, sch := range sbh.History {
			chatQueue, replyChains := sch.ChatQueue, sch.ReplyChains

			// Add local chat queue to jobs
			chatQueue.mu.RLock()
			if !chatQueue.IsShared {
				jobs = append(jobs, CleanJob{
					ChatQueue: chatQueue,
				})
			}
			chatQueue.mu.RUnlock()

			// Add reply chains to jobs
			replyChains.mu.RLock()
			jobs = append(jobs, CleanJob{
				ReplyChains: replyChains,
			})
			replyChains.mu.RUnlock()
		}
		sbh.mu.RUnlock()
	}

	logger.Info(fmt.Sprintf("collected %d jobs", len(jobs)))
	return jobs
}

// Sends jobs to channel with group context
func jobSender(
	gctx context.Context,
	jobs []CleanJob,
	jobsChan chan<- CleanJob,
	logSendingOnce *sync.Once,
	logger *logging.Logger,
) error {
	// CLOSE channel after all sent
	defer close(jobsChan)

	// SEND jobs until group context DONE
	for _, job := range jobs {
		select {
		case jobsChan <- job:
		case <-gctx.Done(): // all workers done
			logSendingOnce.Do(func() { // log only first time
				logger.Info(ctxDoneMsg + opSendingJobs)
			})
			return gctx.Err()
		}
	}

	return nil
}

// Gets clean jobs with group context and performs them
func (h *History) cleanWorker(
	gctx context.Context,
	workerID int,
	jobsChan <-chan CleanJob,
	currentTime time.Time,
	messageTTL time.Duration,
	logGettingOnce *sync.Once,
	logger *logging.Logger,
) error {
	// GET jobs until channel CLOSED or group context DONE
	for processed := 0; ; processed++ {
		select {
		case job, ok := <-jobsChan:
			if !ok {
				logger.Info("jobs channel closed")
				logger.Info(
					fmt.Sprintf("worker %d processed %d jobs",
						workerID, processed,
					),
				)
				return nil
			}

			// Perform clean job
			job.perform(currentTime, messageTTL)

		case <-gctx.Done(): // all workers done
			logGettingOnce.Do(func() { // log only first time
				logger.Info(ctxDoneMsg + opGettingJobs)
			})
			return gctx.Err()
		}
	}
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
	// is longer than time to live
	for line, messageEntry := range replyChains {
		if currentTime.Sub(messageEntry.Timestamp) > messageTTL {
			delete(replyChains, line)
		}
	}
}
