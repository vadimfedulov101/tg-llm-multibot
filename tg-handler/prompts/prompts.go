package prompts

import (
	"fmt"

	"tg-handler/conf"
	"tg-handler/memory"
)

// messaging.MessageInfo abstraction
type Message interface {
	SenderProvider
	LineProvider
}

type SenderProvider interface {
	Sender() string
}

type LineProvider interface {
	Line() string
}

// All prompts
type Prompts struct {
	Response string
	Select   string
	Note     string
	Carma    string
}

// Formats all prompts incrementally
func New(
	conf *conf.BotConf,
	templates *conf.PromptTemplates,
	memory *memory.Memory,
	msg Message,
	botName string,
	chatTitle string,
) *Prompts {
	var (
		// Get templates
		responseTemplate = templates.Response
		selectTemplate   = templates.Select
		noteTemplate     = templates.Note
		carmaTemplate    = templates.Carma

		// Get settings
		candidateNum = conf.Main.CandidateNum
		noteLimit    = memory.Limits.Note

		// Get message data
		userName = msg.Sender()
		line     = msg.Line()

		// Get names
		names = NewNames(botName, userName)
	)

	return &Prompts{
		Response: fmtResponsePrompt(
			responseTemplate, memory, names, chatTitle,
		),
		Select: fmtSelectPrompt(
			selectTemplate, memory, names, candidateNum,
		),
		Note: fmtNotePrompt(
			noteTemplate, memory, names, line, noteLimit,
		),
		Carma: fmtCarmaPrompt(
			carmaTemplate, memory, names, line,
		),
	}
}

// Names representation
type Names struct {
	Bot  string
	User string
}

func NewNames(bot string, user string) *Names {
	return &Names{
		Bot:  bot,
		User: user,
	}
}

// Finalizes candidates prompt formatting (avoid importing the type)
func FinFmtSelectPrompt[T fmt.Stringer](
	prompt string,
	candidates T,
) string {
	return fmt.Sprintf(prompt, candidates)
}

// Finalizes note prompt formatting
func FinFmtNotePrompt(prompt string, botReply string) string {
	return fmt.Sprintf(prompt, botReply)
}

// Finalizes carma prompt formatting
func FinFmtCarmaPrompt(prompt string, botReply string) string {
	return fmt.Sprintf(prompt, botReply)
}

// Formats response prompt
func fmtResponsePrompt(
	template string,
	memory *memory.Memory,
	names *Names,
	chatTitle string,
) string {
	var botName = names.Bot

	return fmt.Sprintf(template,
		botName,
		chatTitle,
		memory.BotContacts,
		memory.ReplyChainLines,
		memory.ChatQueueLines,
		names.Bot,
	)
}

// Formats select prompt incrementally
func fmtSelectPrompt(
	template string,
	memory *memory.Memory,
	names *Names,
	candidateNum int,
) string {
	var botName = names.Bot

	return fmt.Sprintf(template,
		botName,
		memory.BotContacts,
		memory.ChatQueueLines,
		memory.ReplyChainLines,
		"%s", // Response candidates placeholder
		candidateNum,
	)
}

// Formats note prompt incrementally
func fmtNotePrompt(
	template string,
	memory *memory.Memory,
	names *Names,
	line string,
	limit int,
) string {
	var (
		botName  = names.Bot
		userName = names.User
		contact  = memory.BotContacts.Get(userName)
	)

	return fmt.Sprintf(template,
		userName, botName, limit,
		memory.BotContacts,
		memory.ReplyChainLines,
		memory.ChatQueueLines,
		line,
		"%s", // Final response placeholder
		userName, contact.Note,
		userName, limit,
	)

}

// Formats carma prompt incrementally
func fmtCarmaPrompt(
	template string,
	memory *memory.Memory,
	names *Names,
	line string,
) string {
	var (
		botName  = names.Bot
		userName = names.User
		contact  = memory.BotContacts.Get(userName)
	)

	return fmt.Sprintf(template,
		userName, botName,
		memory.BotContacts,
		memory.ReplyChainLines,
		memory.ChatQueueLines,
		line,
		"%s", // Final response placeholder
		userName, contact.Carma,
	)
}
