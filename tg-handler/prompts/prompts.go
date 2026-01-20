package prompts

import (
	"fmt"

	"tg-handler/conf"
	"tg-handler/memory"
)

// Abstract sender provider
type SenderProvider interface {
	Sender() string
}

// Prompts from formatted templates
type Prompts struct {
	Response string
	Select   string
	Tags     string
	Carma    string
}

// Formats all prompts from templates incrementally
func New(
	templates *conf.PromptTemplates,
	memory *memory.Memory,
	senderP SenderProvider,
	botName string,
	chatTitle string,
	candidateNum int,
) *Prompts {
	var (
		// Get templates
		responseTemplate = templates.Response
		selectTemplate   = templates.Select
		tagsTemplate     = templates.Tags
		carmaTemplate    = templates.Carma

		// Get tags limit
		tagsLimit = memory.Limits.Tags

		// Get sender username
		userName = senderP.Sender()
	)

	// Get names
	names := NewNames(botName, userName)

	return &Prompts{
		Response: fmtResponsePrompt(
			responseTemplate, memory, names, chatTitle,
		),
		Select: fmtSelectPrompt(
			selectTemplate, memory, names, candidateNum,
		),
		Tags: fmtTagsPrompt(
			tagsTemplate, memory, names, tagsLimit,
		),
		Carma: fmtCarmaPrompt(
			carmaTemplate, memory, names,
		),
	}
}

// Names type
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

// Finalizes select prompt formatting
func FinFmtSelectPrompt(prompt string, candidates []string) string {
	return fmt.Sprintf(prompt, candidates)
}

// Finalizes tags prompt formatting
func FinFmtTagsPrompt(prompt string, replyLine string) string {
	return fmt.Sprintf(prompt, replyLine)
}

// Finalizes carma prompt formatting
func FinFmtCarmaPrompt(prompt string, replyLine string) string {
	return fmt.Sprintf(prompt, replyLine)
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
		botName, chatTitle, memory, names.Bot,
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
		botName, memory,
		"%s", // Response candidates placeholder
		candidateNum,
	)
}

// Formats tags prompt incrementally
func fmtTagsPrompt(
	template string,
	memory *memory.Memory,
	names *Names,
	lim int,
) string {
	var (
		botName  = names.Bot
		userName = names.User
		contact  = memory.BotContacts.Get(userName)
	)

	return fmt.Sprintf(template,
		userName, botName, memory,
		"%s", // Final response placeholder
		userName, contact.Tags,
		userName, lim,
	)

}

// Formats carma prompt incrementally
func fmtCarmaPrompt(
	template string,
	memory *memory.Memory,
	names *Names,
) string {
	var (
		botName  = names.Bot
		userName = names.User
		contact  = memory.BotContacts.Get(userName)
	)

	return fmt.Sprintf(template,
		userName, botName, memory,
		"%s", // Final response placeholder
		userName, contact.Carma,
	)
}
