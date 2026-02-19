package bot

import (
	"slices"
	"strings"

	tg "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Gets admin identifier for bot
func (bot *Bot) getAdminDetector() func(*tg.Message, string) bool {
	admins := bot.Settings.AllowedChats.Usernames

	// Identifies if private sender is admin
	return func(msg *tg.Message, sender string) bool {
		if msg.Chat.IsPrivate() {
			return slices.Contains(admins, sender)
		}
		return false
	}
}

// Gets reply identifier for bot
func (bot *Bot) getReplyDetector() func(msg *tg.Message) bool {
	return func(msg *tg.Message) bool {
		var isReplied bool

		// Get replied message
		repliedMsg := msg.ReplyToMessage
		// Try to get replied user ID
		var repliedUserID int64
		if repliedMsg != nil {
			repliedUserID = repliedMsg.From.ID
		}

		// Check if bot is replied
		if repliedUserID == bot.ID {
			isReplied = true
		}

		return isReplied
	}
}

// Gets mention identifier for bot
func (bot *Bot) getMentionDetector() func(string) bool {
	// Identifies if text contains bot's @username
	return func(text string) bool {
		return strings.Contains(text, "@"+bot.UserName)
	}
}

// Gets mention modifier for bot
func (bot *Bot) getMentionModifier() func(string) string {
	// Substitutes bot's @username to first name in text
	return func(text string) string {
		return strings.ReplaceAll(text, "@"+bot.UserName, bot.FirstName)
	}
}

// Gets chat validator for bot
func (bot *Bot) getChatValidator() func(int64) bool {
	allowedCIDs := bot.Settings.AllowedChats.IDs

	// Identifies if chat has allowed ID
	return func(cid int64) bool {
		for _, allowedCID := range allowedCIDs {
			if cid == allowedCID {
				return true
			}
		}
		return false
	}
}
