package bot

import (
	"context"
	"log"
	"path/filepath"
	"sync"
	"time"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/api"
	"tg-handler/initconf"
	"tg-handler/memory"
	"tg-handler/messaging"
)

// Cleans bot memory with intervals according to TTL
func Cleaner(
	ctx context.Context,
	sh *memory.SafeHistory,
	historyPath string,
	cleanupInterval time.Duration,
	messageTTL time.Duration,
) {
	// Perform preemptive cleanup
	memory.CleanFileHistory(sh, historyPath, messageTTL)

	// Start a ticker
	t := time.NewTicker(cleanupInterval)
	defer t.Stop()

	// Clean up on tick until context done
	for {
		select {
		case <-t.C:
			memory.CleanFileHistory(sh, historyPath, messageTTL)
		case <-ctx.Done():
			log.Println("Cleaner performing shutdown on signal.")
			return
		}
	}

}

func StartWithCtx(
	ctx context.Context,
	id int,
	initJSON *initconf.InitJSON,
	sh *memory.SafeHistory,
) {
	// Authorize a bot via keyAPI
	bot, err := tg.NewBotAPI(initJSON.KeysAPI[id])
	if err != nil {
		panic(err)
	}
	// Log authorization
	botName := bot.Self.UserName
	log.Printf("Authorized as %s", botName)

	// Get bot history and config (order postfix added by OrderInfo)
	sbh := sh.Get(botName)
	botConf := filepath.Join(initJSON.ConfigPath, botName+"%s.json")

	// Prepare update channel
	u := tg.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	var wg sync.WaitGroup
	for {
		select {
		case update, ok := <-updates:
			if !ok { // If chanel closed await handlers finish before return
				log.Printf("Bot %s update channel closing", botName)
				wg.Wait()
				memory.SaveHistory(initJSON.HistoryPath, sh)
				log.Printf("Bot %s update channel closed", botName)
				return
			}
			// Start handler as awaitable goroutine
			wg.Add(1)
			go func(u tgbotapi.Update) {
				defer wg.Done()
				handleUpdate(ctx, u, initJSON, bot, botName, sbh, botConf)
			}(update)
			// Await handlers finish and save history
			wg.Wait()
			memory.SaveHistory(initJSON.HistoryPath, sh)
		case <-ctx.Done(): // If context done await handlers finish before return
			log.Printf("Bot %s received shutdown signal", botName)
			wg.Wait()
			memory.SaveHistory(initJSON.HistoryPath, sh)
			log.Printf("Bot %s shut down gracefully", botName)
			return
		}
	}
}

// Starts a bot
func handleUpdate(
	ctx context.Context,
	update tgbotapi.Update,
	initJSON *initconf.InitJSON,
	bot *tgbotapi.BotAPI,
	botName string,
	safeBotHistory *memory.SafeBotHistory,
	botConfig string,
) {
	// Get message
	message := update.Message

	// Validate message (get message info)
	messageInfo := messaging.NewMessageInfo(bot, message)
	if !isMsgValid(messageInfo) {
		return
	}

	// Classify message: to bot or not (get order info)
	orderInfo := messaging.NewOrderInfo(
		messageInfo, botConfig, initJSON.Orders[botName],
	)
	isAsked := messaging.IsAsked(orderInfo, initJSON.Admins)

	// Sum up info
	chatInfo := messaging.NewChatInfo(orderInfo, initJSON.MemoryLimit)
	safeChatHistory := safeBotHistory.Get(chatInfo.CID)

	// Record message
	memory.AddToChatContext(
		safeChatHistory, messageInfo, initJSON.MemoryLimit,
	)

	// Gate keep
	if !isAsked {
		return
	}
	log.Printf("%s got message", botName)

	// Respond
	respond(ctx, chatInfo, safeChatHistory)
}

func respond(
	ctx context.Context,
	chat *messaging.ChatInfo,
	sh *memory.SafeChatHistory,
) {
	// Type until reply
	typingCtx, cancel := context.WithCancel(ctx)
	go messaging.Typing(typingCtx, chat)
	defer cancel()

	// Record current and previous messages (get them as lines)
	prev := messaging.NewMessageInfo(chat.Bot, chat.Message.ReplyToMessage)
	lines := memory.AddToReplyChain(sh, [2]memory.IMessage{chat, prev})

	// Reconstruct short/long memory from context/reply chain
	botMemory := memory.GetMemory(sh, lines, chat.MemoryLimit)

	// Send to API
	text, err := api.Send(ctx, botMemory, chat.Config, chat.ChatTitle)
	if err != nil {
		log.Printf("API error in chat %s.", chat.ChatTitle)
		return
	}

	// Reply
	reply := messaging.Reply(chat, text)

	// Record response and preformatted message
	resp := messaging.NewMessageInfo(chat.Bot, reply)
	memory.AddToReplyChainReuse(sh, resp, lines[0])
}

// Validates message based on its info
func isMsgValid(m *messaging.MessageInfo) bool {
	if m == nil {
		return false
	}
	if m.Sender == "" || m.Text == "" {
		return false
	}
	return true
}
