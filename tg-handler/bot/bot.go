package bot

import (
	"context"
	"errors"
	"log"
	"path/filepath"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/conf"
	"tg-handler/history"
	"tg-handler/memory"
	"tg-handler/messaging"
	"tg-handler/model"
)

// Bot errors
var (
	ErrAuth      = errors.New("[bot] authorization failed")
	ErrFailReply = errors.New("[bot] failed to reply")
)

type Bot struct {
	API         *tg.BotAPI
	ID          int64
	UserName    string
	FirstName   string
	Conf        *conf.BotConf            // Bot config
	Settings    *conf.BotSettings        // Init config
	ChatQueues  history.SharedChatQueues // Preinit, shared, r-only
	UpdSignalCh chan<- any               // Signal update end
	History     *history.SafeBotHistory  // Chat histories
	Contacts    *history.SafeBotContacts // Chat agnostic contacts
}

func New(
	keyAPI string,
	iConf *conf.InitConf,
	h *history.History,
	updSignalCh chan<- any,
) *Bot {
	// Authorize as bot
	b, err := tg.NewBotAPI(keyAPI)
	if err != nil {
		log.Panicf("%v: %v", ErrAuth, err)
	}

	// Get bot names
	var (
		userName  = b.Self.UserName
		firstName = b.Self.FirstName
	)
	defer log.Printf("[bot] %s authorized", userName)

	// Get bot history and contacts
	data := h.Bots.Get(userName)
	history, contacts := data.History, data.Contacts

	// Get bot config path
	confPath := filepath.Join(
		iConf.Paths.BotsConfDir, userName+".json",
	)
	// Get bot config
	botConf := conf.MustLoadBotConf(confPath)

	// Return bot instance via pointer
	return &Bot{
		API:         b,
		ID:          b.Self.ID,
		UserName:    userName,
		FirstName:   firstName,
		Conf:        botConf,
		Settings:    &iConf.BotSettings,
		ChatQueues:  h.SharedChatQueues,
		UpdSignalCh: updSignalCh,
		History:     history,
		Contacts:    contacts,
	}
}

// Gets message info for bot
func (bot *Bot) getMessageInfo(
	msg *tg.Message,
) *messaging.MessageInfo {
	return messaging.NewMessageInfo(
		bot.API, msg,
		bot.getAdminDetector(),
		bot.getReplyDetector(),
		bot.getMentionDetector(),
		bot.getMentionModifier(),
		1,
	)
}

// Gets chat info for bot
func (bot *Bot) getChatInfo(
	msgInfo *messaging.MessageInfo,
) *messaging.ChatInfo {
	return messaging.NewChatInfo(
		msgInfo,
		bot.History,
		bot.ChatQueues, // Shared chat queues for public chats
		bot.getChatValidator(),
	)
}

// Starts bot
func (bot *Bot) Start(ctx context.Context) {
	// Prepare updates channel
	u := tg.NewUpdate(0)
	u.Timeout = 30
	updates := bot.API.GetUpdatesChan(u)

	// Handle updates until updates channel CLOSED or context DONE
	defer log.Printf("[bot] %s shut down gracefully", bot.UserName)
	for {
		select {
		case update, ok := <-updates:
			if !ok { // Check if updates channel closed
				log.Printf(
					"[bot] %s update channel closed", bot.UserName,
				)
				return
			}
			bot.handleUpdate(ctx, update)
		case <-ctx.Done():
			log.Printf(
				"[bot] %s received shutdown signal", bot.UserName,
			)
			return
		}
	}
}

// Handles update
func (bot *Bot) handleUpdate(ctx context.Context, upd tg.Update) {
	// Get message info and check if valid
	msgInfo := bot.getMessageInfo(upd.Message)
	if msgInfo == nil {
		return
	}

	// Get chat info and check if allowed
	chatInfo := bot.getChatInfo(msgInfo)
	if !chatInfo.IsAllowed {
		return
	}

	// Check if not triggered
	if !chatInfo.LastMsg.IsTriggering {
		// Save message anyway
		chatInfo.History.AddToChatQueue(chatInfo.LastMsg)
		return
	}

	bot.handleMessage(ctx, chatInfo)
}

// Handles message in chat context
func (bot *Bot) handleMessage(
	ctx context.Context,
	chatInfo *messaging.ChatInfo,
) {
	log.Printf("[bot] %s got message", bot.UserName)

	// Get memory
	memory := memory.New(
		chatInfo.History, bot.Contacts,
		chatInfo.LastMsg, &bot.Settings.MemoryLimits,
	)

	// Create model
	model := model.New(
		bot.Conf, &bot.Settings.PromptTemplates, memory,
		chatInfo.LastMsg, bot.FirstName, chatInfo.Title,
	)

	// Add message to history and proceed in separate goroutine
	chatInfo.History.AddToBoth(chatInfo.LastMsg)
	go func() {
		// Reply
		replyInfo := bot.reply(ctx, model, chatInfo)
		chatInfo.History.AddToBoth(replyInfo)

		// Reflect on reply
		chatInfo.LastMsg = replyInfo
		bot.reflect(ctx, model, chatInfo)

		// Send update signal
		bot.UpdSignalCh <- struct{}{}
	}()
}

// Replies to message in chat, return reply message info
func (bot *Bot) reply(
	ctx context.Context,
	model *model.Model,
	chatInfo *messaging.ChatInfo,
) *messaging.MessageInfo {
	var replyInfo *messaging.MessageInfo

	// Type until reply
	typingCtx, cancel := context.WithCancel(ctx)
	go messaging.Type(typingCtx, bot.API, chatInfo)
	defer cancel()

	// React via model
	text := model.React(ctx)

	// Reply
	reply := messaging.Reply(bot.API, chatInfo, text)
	replyInfo = bot.getMessageInfo(reply)

	return replyInfo
}

// Reflects on its reply
func (bot *Bot) reflect(
	ctx context.Context,
	model *model.Model,
	chatInfo *messaging.ChatInfo,
) {
	model.Reflect(ctx, chatInfo.LastMsg)
}
