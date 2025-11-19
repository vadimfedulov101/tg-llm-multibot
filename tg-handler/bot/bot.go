package bot

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/api"
	"tg-handler/initconf"
	"tg-handler/memory"
	"tg-handler/messaging"
)

func StartWithCtx(
	ctx context.Context,
	id int,
	initConf *initconf.InitConf,
	sh *memory.SafeHistory,
) {
	// Authorize a bot via keyAPI
	bot, err := tg.NewBotAPI(initConf.KeysAPI[id])
	if err != nil {
		panic(err)
	}
	// Log authorization
	botName := bot.Self.UserName
	log.Printf("Authorized as %s", botName)

	// Get bot history and config (order postfix added by OrderInfo)
	sbh := sh.Get(botName)
	botConf := filepath.Join(initConf.ConfigPath, botName+"%s.json")

	// Prepare update channel
	u := tg.NewUpdate(0)
	u.Timeout = 30
	updates := bot.GetUpdatesChan(u)

	for {
		select {
		case update, ok := <-updates:
			if !ok { // If chanel closed await handlers finish before return
				log.Printf("Bot %s update channel closing", botName)
				memory.SaveHistory(initConf.HistoryPath, sh)
				log.Printf("Bot %s update channel closed", botName)
				return
			}
			// Start handler as awaitable goroutine
			go func(u tgbotapi.Update) {
				handleUpdate(ctx, u, initConf, bot, botName, sbh, botConf)
				memory.SaveHistory(initConf.HistoryPath, sh)
			}(update)
		case <-ctx.Done(): // If context done await handlers finish before return
			log.Printf("Bot %s received shutdown signal", botName)
			memory.SaveHistory(initConf.HistoryPath, sh)
			log.Printf("Bot %s shut down gracefully", botName)
			return
		}
	}
}

// Starts a bot
func handleUpdate(
	ctx context.Context,
	update tgbotapi.Update,
	initConf *initconf.InitConf,
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
		messageInfo, botConfig, initConf.Orders[botName],
	)
	isAsked := messaging.IsAsked(orderInfo, initConf.Admins)

	// Sum up info
	chatInfo := messaging.NewChatInfo(orderInfo, initConf.MemoryConfig.Limit)
	safeChatHistory := safeBotHistory.Get(chatInfo.CID)

	// Record message
	memory.AddToChatContext(
		safeChatHistory, messageInfo, initConf.MemoryConfig.Limit,
	)

	// Gate keep
	if !isAsked {
		return
	}
	log.Printf("%s got message", botName)

	// Respond and fail fatally if could not do that
	err := respond(
		ctx, chatInfo, safeChatHistory, &initConf.Prompts, initConf.CandidateNum,
	)
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func respond(
	ctx context.Context,
	chat *messaging.ChatInfo,
	sh *memory.SafeChatHistory,
	prompts *initconf.Prompts,
	candidateNum int,
) error {
	// Type until reply
	typingCtx, cancel := context.WithCancel(ctx)
	go messaging.Typing(typingCtx, chat)
	defer cancel()

	// Record current and previous messages (get them as lines)
	prev := messaging.NewMessageInfo(chat.Bot, chat.Message.ReplyToMessage)
	lines := memory.AddToReplyChain(sh, [2]memory.IMessage{chat, prev})

	// Reconstruct short/long memory from context/reply chain
	botMemory := memory.GetMemory(sh, lines, chat.MemoryLimit)

	// Generate response
	text, err := api.Generate(
		ctx, chat.Config, chat.ChatTitle, botMemory, prompts, candidateNum,
	)
	if err != nil {
		return fmt.Errorf("Failed to respond in chat %s: %w", chat.ChatTitle, err)
	}

	// Reply with response
	reply := messaging.Reply(chat, text)

	// Record reply and preformatted message
	resp := messaging.NewMessageInfo(chat.Bot, reply)
	memory.AddToReplyChainReuse(sh, resp, lines[0])

	return nil
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
