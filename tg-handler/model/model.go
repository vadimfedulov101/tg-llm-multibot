package model

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"tg-handler/carma"
	"tg-handler/conf"
	"tg-handler/memory"
	"tg-handler/prompts"
	"tg-handler/tags"
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

// Model errors
var (
	ErrSelectFailed      = errors.New("[model] select failed")
	ErrCarmaUpdateFailed = errors.New(
		"[model] carma update failed",
	)
)

// Constants
const (
	apiUrl       = "http://ollama:11434/api/generate"
	retryTime    = time.Minute
	waitTimeout  = 10 * time.Minute
	maxSelectTry = 10
	maxTagsTry   = 10
	maxCarmaTry  = 10
)

// LLM model
type Model struct {
	Config    *conf.BotConf
	Prompts   *prompts.Prompts
	Memory    *memory.Memory
	BotName   string
	ChatTitle string
}

func New(
	botConf *conf.BotConf,
	promptTemplates *conf.PromptTemplates,
	memory *memory.Memory,
	lastMsg Message,
	botName string,
	chatTitle string,
) *Model {
	// Format prompts
	prompts := prompts.New(
		promptTemplates, memory, lastMsg, botName, chatTitle,
	)

	return &Model{
		Config:    botConf,
		Prompts:   prompts,
		Memory:    memory,
		BotName:   botName,
		ChatTitle: chatTitle,
	}
}

// Reacts to new message
func (m *Model) React(ctx context.Context) string {
	candidates := m.generateCandidates(ctx)
	bestCandidate := m.selectBestCandidate(ctx, candidates)
	return bestCandidate
}

// Reflects on response
func (m *Model) Reflect(
	ctx context.Context,
	msg Message,
) {
	var (
		sender      = msg.Sender()
		botContacts = m.Memory.BotContacts
	)

	// Get bot contact to update
	botContact := botContacts.Get(sender)

	// Update carma
	carmaUpdate := m.updateCarma(ctx, msg.Line())
	botContact.Carma.Apply(carmaUpdate)

	// Update persona
	tags := m.updateTags(ctx, msg.Line())
	botContact.Tags = tags

	// Reset bot contacts
	botContacts.Set(sender, botContact)
}

// Generates candidates
func (m *Model) generateCandidates(ctx context.Context) []string {
	var (
		candidateNum = m.Config.Main.CandidateNum
		candidates   = make([]string, 0, candidateNum)
	)

	// Form request
	request := newRequest(m.Prompts.Response, m.Config)

	// Generate candidates
	for i := range candidateNum {
		// Get new candidate
		candidate := sendRequestEternal(ctx, request)
		log.Printf("[model] candidate %d: %s", i+1, candidate)

		// Append to candidates
		candidates = append(candidates, candidate)
	}

	return candidates
}

// Select the best candidate
func (m *Model) selectBestCandidate(
	ctx context.Context,
	candidates Candidates,
) string {
	// One candidate to be selected, return it
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Format prompt
	prompt := prompts.FinFmtSelectPrompt(
		m.Prompts.Select, candidates,
	)
	// Form request
	request := newRequest(prompt, m.Config)

	// Try to select the best candidate
	for i := range maxSelectTry {
		// Log start
		tryStr := fmt.Sprintf("select try: %d", i+1)
		log.Printf("%s [%s]", tryStr, "...")

		// Try to get select index
		selectStr := sendRequestEternal(ctx, request)
		selectIdx, err := newSelectIdx(selectStr, len(candidates))

		// Log success, return candidate
		if err == nil {
			log.Println("%s [success]", tryStr)
			return candidates[selectIdx]
		}

		// Log failure
		log.Printf("%s: [%v: %v]", tryStr, ErrSelectFailed, err)
	}

	// Return first candidate
	return candidates[0]
}

// Generates new tags
func (m *Model) updateTags(
	ctx context.Context,
	line string,
) tags.Tags {
	// Format prompt
	prompt := prompts.FinFmtTagsPrompt(m.Prompts.Tags, line)
	// Form request
	request := newRequest(prompt, m.Config)

	// Get tags
	tagsStr := sendRequestEternal(ctx, request)
	tags := tags.New(tagsStr, m.Memory.Limits.Tags)

	log.Printf("[model] tags: %s", tags)
	return tags
}

// Generates carma update
func (m *Model) updateCarma(
	ctx context.Context,
	line string,
) carma.Update {
	// Format prompt
	prompt := prompts.FinFmtCarmaPrompt(m.Prompts.Carma, line)
	// Form request
	request := newRequest(prompt, m.Config)

	for i := range maxCarmaTry {
		// Log start
		tryStr := fmt.Sprintf("carma update try: %d", i+1)
		log.Printf("%s [%s]", tryStr, "...")

		// Try to get carma update
		carmaUpdateStr := sendRequestEternal(ctx, request)
		carmaUpdate, err := carma.NewUpdate(carmaUpdateStr)

		// Log success, return carma update
		if err == nil {
			log.Printf("[model] carma update: %s", carmaUpdate)
			return carmaUpdate
		}

		// Log failure
		log.Printf(
			"%s: [%v: %v]", tryStr, ErrCarmaUpdateFailed, err,
		)
	}

	// Return neutral carma update
	log.Printf("[model] carma update: %s", carma.UpdateNeutral)
	return carma.UpdateNeutral
}
