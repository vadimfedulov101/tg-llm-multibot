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
	API                *tg.BotAPI
	ID                 int64
	UserName           string
	FirstName          string
	confPath           string
	Conf               *conf.BotConf
	PromptTemplates    *conf.PromptTemplates
	AllowedChats       *conf.AllowedChats
	Contacts           *history.SafeBotContacts
	HistoryUpdSignalCh chan<- any
	History            *history.SafeBotHistory
	MemoryLimits       *conf.MemoryLimits
}

func New(
	keyAPI string,
	iConf *conf.InitConf,
	sh *history.SafeHistory,
	historyUpdSignalCh chan<- any,
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

	// Get _safe_ bot history and contacts
	history, contacts := sh.Get(userName).Unpack()

	// Get bot config path
	confPath := filepath.Join(iConf.Paths.Bots, userName+".json")
	// Load bot config
	conf := conf.MustLoadBotConf(confPath)

	// Get bot instance as pointer
	return &Bot{
		API:                b,
		ID:                 b.Self.ID,
		UserName:           userName,
		FirstName:          firstName,
		confPath:           confPath,
		Conf:               conf,
		PromptTemplates:    &iConf.PromptTemplates,
		AllowedChats:       &iConf.AllowedChats,
		MemoryLimits:       &iConf.MemoryLimits,
		HistoryUpdSignalCh: historyUpdSignalCh,
		History:            history,
		Contacts:           contacts,
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

	// Defer gracefull shutdown
	defer func() {
		log.Printf("Bot %s shut down gracefully", bot.UserName)
	}()
	// Handle updates until updates end or context done
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				log.Printf(
					"Bot %s update channel closed", bot.UserName,
				)
				return
			}
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
		chatInfo.LastMsg, bot.MemoryLimits,
	)

	// Create model
	model := model.New(
		bot.Conf, bot.PromptTemplates, memory,
		chatInfo.LastMsg, bot.FirstName, chatInfo.Title,
	)

	// Reply with history recording
	replyInfo := bot.reply(ctx, model, chatInfo)

	// Reflect on reply
	chatInfo.LastMsg = replyInfo
	bot.reflect(ctx, model, chatInfo)

	// Save safe history
	bot.HistoryUpdSignalCh <- struct{}{}
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
