package bot

import (
	"context"
	"path/filepath"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/api"
	"tg-handler/conf"
	"tg-handler/history"
	"tg-handler/messaging"
)

// Replies to message in chat info
func reply(
	ctx context.Context,
	chatInfo *messaging.ChatInfo,
	iConf *conf.InitConf,
	safeChatHistory *history.SafeChatHistory,
	bot *tg.BotAPI,
	botName string,
) *messaging.MessageInfo {
	// Type until reply
	typingCtx, cancel := context.WithCancel(ctx)
	go messaging.Type(typingCtx, bot, chatInfo)
	defer cancel()

	// Get memory
	memoryLimit := iConf.GenerateConf.MemoryLimit
	memory := history.NewMemory(
		safeChatHistory, &chatInfo.MessageInfo, memoryLimit,
	)

	// Generate reply with specific bot config
	confPath := filepath.Join(iConf.PathsConf.Bots, botName+".json")
	text := api.Generate(
		ctx, memory, confPath, &iConf.GenerateConf.Prompts, bot.Self.FirstName, chatInfo.ChatTitle,
	)

	// Reply
	reply := messaging.Reply(bot, chatInfo, text)
	replyInfo := messaging.NewMessageInfo(bot, reply, 1)

	return replyInfo
}
