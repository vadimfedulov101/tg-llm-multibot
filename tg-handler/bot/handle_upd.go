package bot

import (
	"context"
	"log"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"tg-handler/conf"
	"tg-handler/history"
	"tg-handler/messaging"
)

// Handles separate update
func handleUpdate(
	ctx context.Context,
	update tg.Update,
	iConf *conf.InitConf,
	safeHistory *history.SafeHistory,
	bot *tg.BotAPI,
	botName string,
) {
	// Get safe bot history
	safeBotHistory := safeHistory.Get(botName)

	// Get and validate message info
	msgInfo := messaging.NewMessageInfo(bot, update.Message, 1)
	if msgInfo == nil || msgInfo.Text == "" {
		return
	}

	// Get chat info and react
	chatInfo := messaging.NewChatInfo(msgInfo)
	isAsked := chatInfo.IsAsked(bot, iConf.BotsConf.Admins, iConf.BotsConf.CIDs)
	if !isAsked {
		return
	}
	log.Printf("[bot] %s got message", botName)

	// Get safe chat history
	safeChatHistory := safeBotHistory.Get(chatInfo.CID)

	// Reply with history recording
	safeChatHistory.AddTo(&chatInfo.MessageInfo)
	replyInfo := reply(ctx, chatInfo, iConf, safeChatHistory, bot, botName)
	safeChatHistory.AddTo(replyInfo)

	// Save safe history
	safeHistory.Save(iConf.PathsConf.History)
}
