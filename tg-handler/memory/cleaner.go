package memory

import (
	"context"
	"log"
	"time"

	"tg-handler/initconf"
)

// Cleans bot memory with intervals according to TTL
func Cleaner(
	ctx context.Context,
	sh *SafeHistory,
	historyPath string,
	memConf *initconf.MemoryConfig,
) {
	// Perform preemptive cleanup
	cleanFileHistory(sh, historyPath, memConf.MessageTTL)

	// Start a ticker
	t := time.NewTicker(memConf.CleanupInterval)
	defer t.Stop()

	// Clean up on tick until context done
	for {
		select {
		case <-t.C:
			cleanFileHistory(sh, historyPath, memConf.MessageTTL)
		case <-ctx.Done():
			log.Println("Cleaner performing shutdown on signal.")
			return
		}
	}
}

// Cleans file history (clean local and save as file)
func cleanFileHistory(sh *SafeHistory, dest string, messageTTL time.Duration) {
	cleanHistory(sh, messageTTL)
	if err := SaveHistory(dest, sh); err != nil {
		log.Printf("Failed to save history: %v", err)
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

	currentTime := time.Now()

	// Clean each bot independently
	for _, botName := range botNames {
		safeBotHistory := sh.Get(botName)
		if safeBotHistory != nil {
			cleanBotHistory(safeBotHistory, currentTime, messageTTL)
		}
	}
}

func cleanBotHistory(sbh *SafeBotHistory, currentTime time.Time, messageTTL time.Duration) {
	// Ensure secure access
	sbh.mu.RLock()
	CIDs := make([]int64, 0, len(sbh.History))
	for CID := range sbh.History {
		CIDs = append(CIDs, CID)
	}
	sbh.mu.RUnlock()

	for _, CID := range CIDs {
		safeChatHistory := sbh.Get(CID)
		cleanChatHistory(safeChatHistory, currentTime, messageTTL)

	}
}

func cleanChatHistory(sch *SafeChatHistory, currentTime time.Time, messageTTL time.Duration) {
	// Ensure secure access
	sch.mu.Lock()
	defer sch.mu.Unlock()

	// Clean up context
	// Create a new slice pointing to the original slice
	filteredContext := sch.History.ChatContext[:0]
	// Range over the original slice reusing its memory for a new slice
	for _, messageEntry := range sch.History.ChatContext {
		if currentTime.Sub(messageEntry.Timestamp) <= messageTTL {
			filteredContext = append(filteredContext, messageEntry)
		}
	}

	// Clean up reply chains
	for line, messageEntry := range sch.History.ReplyChains {
		if currentTime.Sub(messageEntry.Timestamp) > messageTTL {
			delete(sch.History.ReplyChains, line)
		}
	}
}
