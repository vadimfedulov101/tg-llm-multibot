package history

import (
	"context"
	"errors"
	"log"
	"time"

	"tg-handler/conf"
)

// Cleaner errors
var (
	ErrSaveFailed = errors.New("[cleaner] failed to save history")
)

// Cleans history with intervals according to TTL
func Cleaner(
	ctx context.Context,
	historyPath string,
	sh *SafeHistory,
	conf *conf.CleanerConf,
) {
	// Get variables
	var (
		cleanupInterval = time.Duration(conf.CleanupInterval)
		messageTTL      = time.Duration(conf.MessageTTL)
	)

	// Clean up in advance
	cleanFileHistory(sh, historyPath, messageTTL)

	// Start ticker
	t := time.NewTicker(cleanupInterval)
	defer t.Stop()

	// Clean up on tick until context done
	defer log.Println("Cleaner shut down gracefully.")
	for {
		select {
		case <-t.C:
			cleanFileHistory(sh, historyPath, messageTTL)
		case <-ctx.Done():
			log.Println("Cleaner received shutdown signal.")
			return
		}
	}
}

// Cleans file history (clean local and save as file)
func cleanFileHistory(
	sh *SafeHistory,
	historyPath string,
	messageTTL time.Duration,
) {
	cleanHistory(sh, messageTTL)
	if err := sh.Save(historyPath); err != nil {
		log.Printf("%v: %v", ErrSaveFailed, err)
	}
}

// Cleans all lines older than day in every chat history
func cleanHistory(sh *SafeHistory, messageTTL time.Duration) {
	// Get all bot names
	sh.mu.RLock()
	botNames := make([]string, 0, len(sh.History))
	for botName := range sh.History {
		botNames = append(botNames, botName)
	}
	sh.mu.RUnlock()

	// Get current time
	currentTime := time.Now()

	// Clean each bot independently
	for _, botName := range botNames {
		safeBotHistory := sh.Get(botName)
		if safeBotHistory != nil {
			cleanBotHistory(safeBotHistory, currentTime, messageTTL)
		}
	}
}

// Cleans bot history
func cleanBotHistory(
	sbh *SafeBotHistory,
	currentTime time.Time,
	messageTTL time.Duration,
) {
	// Get all chat IDs
	sbh.mu.RLock()
	CIDs := make([]int64, 0, len(sbh.History))
	for CID := range sbh.History {
		CIDs = append(CIDs, CID)
	}
	sbh.mu.RUnlock()

	// Clean each chat independently
	for _, CID := range CIDs {
		safeChatHistory := sbh.Get(CID)
		cleanChatHistory(safeChatHistory, currentTime, messageTTL)
	}
}

// Cleans chat history
func cleanChatHistory(
	sch *SafeChatHistory,
	currentTime time.Time,
	messageTTL time.Duration,
) {
	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	sch.History.ChatContext.clean(currentTime, messageTTL)
	sch.History.ReplyChain.clean(currentTime, messageTTL)
}

// Cleans chat context based on expiration
func (c *ChatContext) clean(currentTime time.Time, messageTTL time.Duration) {
	// Create a new slice pointing to the same array
	chatContext := (*c)[:0]
	// Append every not-expired message entry to a new slice
	for _, messageEntry := range *c {
		if currentTime.Sub(messageEntry.Timestamp) <= messageTTL {
			chatContext = append(chatContext, messageEntry)
		}
	}
	// Set array to new slice
	*c = chatContext
}

// Cleans reply chains based on expiration
func (r *ReplyChain) clean(currentTime time.Time, messageTTL time.Duration) {
	// Delete every expired message entry
	for line, messageEntry := range *r {
		if currentTime.Sub(messageEntry.Timestamp) > messageTTL {
			delete(*r, line)
		}
	}
}
