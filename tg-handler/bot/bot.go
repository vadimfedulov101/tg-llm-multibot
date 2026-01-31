package bot

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/conf"
	"tg-handler/history"
	"tg-handler/logging"
	"tg-handler/memory"
	"tg-handler/messaging"
	"tg-handler/model"
	"tg-handler/names"
	"tg-handler/prompts"
)

// Bot errors
var (
	errAuthFailed     = errors.New("authorization failed")
	errChatNotAllowed = errors.New("chat is not allowed")
	errMsgMalformed   = errors.New("message malformed")
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
	wg          *sync.WaitGroup
	logger      *logging.Logger
}

func New(
	apiKey string,
	iConf *conf.InitConf,
	h *history.History,
	updSignalCh chan<- any,
	wg *sync.WaitGroup,
	logger *logging.Logger,
) *Bot {
	// Authorize as bot
	bot, err := tg.NewBotAPI(apiKey)
	if err != nil {
		logger.With(logging.ApiKey(apiKey)).
			Panic("not authorized", logging.Err(errAuthFailed))
	}

	// Get user name
	var userName = bot.Self.UserName

	// Set logger
	logger = logger.With(logging.BotName(userName))
	defer logger.Info("authorized")

	// Get history and contacts
	data := h.Bots.Get(userName)
	history, contacts := data.History, data.Contacts

	// Get config path
	confPath := filepath.Join(
		iConf.Paths.BotsConfDir, userName+".json",
	)
	// Get config
	botConf := conf.MustLoadBotConf(
		confPath,
		&iConf.BotSettings.DefaultOptions,
		logger,
	)

	return &Bot{
		API:         bot,
		ID:          bot.Self.ID,
		UserName:    userName,
		FirstName:   bot.Self.FirstName,
		Conf:        botConf,
		Settings:    &iConf.BotSettings,
		ChatQueues:  h.SharedChatQueues,
		UpdSignalCh: updSignalCh,
		History:     history,
		Contacts:    contacts,
		wg:          wg,
		logger:      logger,
	}
}

// Starts bot
func (bot *Bot) Start(ctx context.Context) {
	// Prepare updates channel
	u := tg.NewUpdate(0)
	u.Timeout = 30
	updates := bot.API.GetUpdatesChan(u)

	// Handle updates until channel CLOSED or context DONE
	defer bot.logger.Info("shut down gracefully")
	for {
		select {
		case update, ok := <-updates:
			if !ok {
				bot.logger.Info("update channel closed")
				return
			}
			bot.handleUpdate(ctx, update)
		case <-ctx.Done():
			bot.logger.Info("received shutdown signal")
			return
		}
	}
}

// Handles update
func (bot *Bot) handleUpdate(ctx context.Context, upd tg.Update) {
	const errMsg = "update not handled"
	logger := bot.logger

	logger.Info("got update")

	// Get message info and check if valid
	msgInfo, err := bot.getMessageInfo(upd.Message)
	if err != nil {
		logger.Error(errMsg, logging.Err(
			fmt.Errorf("%w: %v", errMsgMalformed, err),
		))
		return
	}

	// Get chat info and check if allowed
	chatInfo := bot.getChatInfo(msgInfo)
	logger = logger.With(logging.ChatID(chatInfo.ID))
	logger = logger.With(logging.UserName(msgInfo.Sender()))
	if !chatInfo.IsAllowed {
		logger.Error(errMsg, logging.Err(errChatNotAllowed))
		return
	}

	// Safe to chat queue if not triggered
	if !chatInfo.LastMsg.IsTriggering {
		chatInfo.History.AddToChatQueue(
			chatInfo.LastMsg, logger,
		)
		return
	}

	bot.handleMessage(ctx, chatInfo, logger)
}

// Handles message in chat context
func (bot *Bot) handleMessage(
	ctx context.Context,
	chatInfo *messaging.ChatInfo,
	logger *logging.Logger,
) {
	const errMsg = "message not handled"

	logger.Info("got message")

	// Add new message to history
	chatInfo.History.AddToBoth(chatInfo.LastMsg, logger)

	// Create names
	names := names.New(bot.FirstName, chatInfo.LastMsg.Sender())

	// Create memory
	memory := memory.New(
		chatInfo.History, bot.Contacts,
		chatInfo.LastMsg, &bot.Settings.MemoryLimits, logger,
	)

	// Get prompts
	prompts := prompts.New(
		&bot.Settings.PromptTemplates,
		memory, names, chatInfo.Title,
		bot.Conf.Main.CandidateNum,
	)

	// Create model
	model := model.New(
		bot.Conf, prompts, memory, names, chatInfo.Title, logger,
	)

	bot.wg.Go(func() {
		// Reply as bot with valid info
		replyInfo, err := bot.reply(ctx, model, chatInfo)
		if err != nil {
			logger.Error(errMsg, logging.Err(err))
			return
		}

		// Add reply to history
		chatInfo.History.AddToBoth(replyInfo, logger)

		// Reflect on reply as model
		err = model.Reflect(
			ctx, chatInfo.LastMsg.Sender(), replyInfo,
		)
		if err != nil {
			logger.Error(errMsg, logging.Err(err))
			return
		}

		// Send update signal
		bot.UpdSignalCh <- struct{}{}
	})
}

// Replies to message in chat, return reply message info
func (bot *Bot) reply(
	ctx context.Context,
	model *model.Model,
	chatInfo *messaging.ChatInfo,
) (*messaging.MessageInfo, error) {
	var replyInfo *messaging.MessageInfo
	var err error

	// Type until reply
	typingCtx, cancel := context.WithCancel(ctx)
	go messaging.Type(typingCtx, bot.API, chatInfo, model.Logger)
	defer cancel()

	// Reply as model
	text, err := model.Reply(ctx)
	if err != nil {
		return nil, err
	}

	// Reply as bot
	reply := messaging.Reply(bot.API, chatInfo, text, model.Logger)
	replyInfo, err = bot.getMessageInfo(reply)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errMsgMalformed, err)
	}

	return replyInfo, nil
}

// Gets message info for bot
func (bot *Bot) getMessageInfo(
	msg *tg.Message,
) (*messaging.MessageInfo, error) {
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
