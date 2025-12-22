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
	ErrAuth = errors.New("[bot] authorization failed")
)

type Bot struct {
	API         *tg.BotAPI
	ID          int64
	UserName    string
	FirstName   string
	Conf        *conf.BotConf     // Loaded from bot config
	Settings    *conf.BotSettings // Loaded from init config
	UpdSignalCh chan<- any
	History     *history.SafeBotHistory
	Contacts    *history.SafeBotContacts
}

func New(
	keyAPI string,
	iConf *conf.InitConf,
	sh *history.SafeHistory,
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
	defer log.Printf("Authorized as %s", userName)

	// Get _safe_ bot data
	data, _ := sh.Get(userName)
	history, contacts := data.Get()

	// Get bot config path
	confPath := filepath.Join(
		iConf.Paths.BotsConfDir, userName+".json",
	)
	// Load bot config
	conf := conf.MustLoadBotConf(confPath)

	// Return bot instance via pointer
	return &Bot{
		API:         b,
		ID:          b.Self.ID,
		UserName:    userName,
		FirstName:   firstName,
		Conf:        conf,
		Settings:    &iConf.BotSettings,
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
		bot.getSenderValidator(),
		bot.getReplyDetector(),
		bot.getMentionDetector(),
		bot.getMentionHumanizer(),
		1,
	)
}

// Gets chat info for bot
func (bot *Bot) getChatInfo(
	msgInfo *messaging.MessageInfo,
) *messaging.ChatInfo {
	return messaging.NewChatInfo(
		msgInfo, bot.History, bot.getChatValidator(),
	)
}

// Starts bot
func (bot *Bot) Start(ctx context.Context) {
	// Prepare updates channel
	u := tg.NewUpdate(0)
	u.Timeout = 30
	updates := bot.API.GetUpdatesChan(u)

	// HANDLE updates until updates channel CLOSED or context DONE
	defer log.Printf("Bot %s shut down gracefully", bot.UserName)
	for {
		select {
		case update, ok := <-updates:
			if !ok { // Check if updates channel closed
				log.Printf(
					"Bot %s update channel closed", bot.UserName,
				)
				return
			}
			// Proceed with handling in separate goroutine
			go bot.handleUpdate(ctx, update)
		case <-ctx.Done():
			log.Printf(
				"Bot %s received shutdown signal", bot.UserName,
			)
			return
		}
	}
}

// Handles update
func (bot *Bot) handleUpdate(ctx context.Context, upd tg.Update) {
	// Get message info and check
	msgInfo := bot.getMessageInfo(upd.Message)
	if msgInfo == nil || !msgInfo.IsTriggering {
		return
	}

	// Get chat info and check
	chatInfo := bot.getChatInfo(msgInfo)
	if !chatInfo.IsAllowed {
		return
	}

	// Handle message
	bot.handleMessage(ctx, chatInfo)

}

// Handles message in chat context
func (bot *Bot) handleMessage(
	ctx context.Context,
	chatInfo *messaging.ChatInfo,
) {
	log.Printf("[bot] %s got message", bot.UserName)

	// Reload bot config
	bot.mustReloadBotConf()

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

	// Reply with history recording
	replyInfo := bot.reply(ctx, model, chatInfo)

	// Reflect on reply
	chatInfo.LastMsg = replyInfo
	bot.reflect(ctx, model, chatInfo)

	// Send update signal
	bot.UpdSignalCh <- struct{}{}
}

// Replies to message in chat
func (bot *Bot) reply(
	ctx context.Context,
	model *model.Model,
	chatInfo *messaging.ChatInfo,
) *messaging.MessageInfo {
	var replyInfo *messaging.MessageInfo

	// Add last and reply messages to memory
	chatInfo.History.AddTo(chatInfo.LastMsg)
	defer chatInfo.History.AddTo(replyInfo)

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
