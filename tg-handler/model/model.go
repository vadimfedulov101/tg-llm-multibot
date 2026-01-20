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
	"tg-handler/selectIdx"
	"tg-handler/tags"
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

// Message abstraction
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
	errGenFailed = errors.New("generation failed")
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
	var candidateNum = botConf.Main.CandidateNum

	// Format prompts from templates
	prompts := prompts.New(
		promptTemplates,
		memory, lastMsg, botName, chatTitle, candidateNum,
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
	candidates := m.genCandidates(ctx)
	bestCandidate := m.selectBestCandidate(ctx, candidates)
	return bestCandidate
}

// Reflects on response
func (m *Model) Reflect(
	ctx context.Context,
	user string,
	reply Message,
) {
	var (
		botContacts = m.Memory.BotContacts
	)

	// Get contact to update
	botContact := botContacts.Get(user)

	// Update carma
	carmaUpdate := m.genCarmaUpdate(ctx, reply.Line())
	botContact.Carma.Apply(carmaUpdate)

	// Update persona
	tags := m.genTags(ctx, reply.Line())
	botContact.Tags = tags

	// Reset contacts
	botContacts.Set(user, botContact)
}

// Generates candidates
func (m *Model) genCandidates(ctx context.Context) []string {
	var (
		candidateNum = m.Config.Main.CandidateNum
		candidates   = make([]string, 0, candidateNum)
	)

	// Form request
	request := newRequest(m.Prompts.Response, m.Config)

	// Generate candidates
	for i := range candidateNum {
		tryStr := fmt.Sprintf("[model] generate (iter %d)", i+1)

		// Log start
		log.Printf("%s: %s", tryStr, "...")

		// Get new candidate
		candidate := sendRequestEternal(ctx, request)

		// Append to candidates
		candidates = append(candidates, candidate)

		// Log successs
		log.Printf("[model] candidate %d: %s", i+1, candidate)
	}

	return candidates
}

// Select the best candidate
func (m *Model) selectBestCandidate(
	ctx context.Context,
	candidates Candidates,
) string {
	const genType = "select index"

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
		tryStr := fmt.Sprintf("[model] select (try %d)", i+1)

		// Log start
		log.Printf("%s: %s", tryStr, "...")

		// Try to get select index
		selectStr := sendRequestEternal(ctx, request)
		selectIdx, err := selectIdx.New(selectStr, len(candidates))

		// Log success, return
		if err == nil {
			log.Printf("%s: %s", tryStr, selectIdx)
			return candidates[selectIdx]
		}

		// Log failure, continue
		log.Printf(
			"%s: %v: %s: %v", tryStr, errGenFailed, genType, err,
		)
	}

	// Fall back
	log.Println("[model] using fallback value for candidates")
	return candidates.Fallback()
}

// Generates unique tags
func (m *Model) genTags(
	ctx context.Context,
	replyLine string,
) tags.Tags {
	const genType = "tags"

	// Format prompt
	prompt := prompts.FinFmtTagsPrompt(m.Prompts.Tags, replyLine)
	// Form request
	request := newRequest(prompt, m.Config)

	for i := range maxTagsTry {
		tagsStr := fmt.Sprintf("[model] tags (try %d)", i+1)

		// Log start
		log.Printf("%s: %s", tagsStr, "...")

		// Get tags
		rawTags := sendRequestEternal(ctx, request)
		tags, err := tags.New(rawTags, m.Memory.Limits.Tags)

		// Log success, return
		if err == nil {
			log.Printf("[model] tags: %s", tags)
			return tags
		}

		// Log failure, continue
		log.Printf(
			"%s: %v: %s: %v", tagsStr, errGenFailed, genType, err,
		)
	}

	// Fall back
	log.Println("[model] using fallback value for tags")
	return tags.Fallback()
}

// Generates carma update
func (m *Model) genCarmaUpdate(
	ctx context.Context,
	replyLine string,
) carma.Update {
	const genType = "carma update"

	// Format prompt
	prompt := prompts.FinFmtCarmaPrompt(m.Prompts.Carma, replyLine)
	// Form request
	request := newRequest(prompt, m.Config)

	for i := range maxCarmaTry {
		tryStr := fmt.Sprintf("[model] carma update (try %d)", i+1)

		// Log start
		log.Printf("%s: %s", tryStr, "...")

		// Try to get carma update
		carmaUpdateStr := sendRequestEternal(ctx, request)
		carmaUpdate, err := carma.NewUpdate(carmaUpdateStr)

		// Log success, return
		if err == nil {
			log.Printf("[model] carma update: %s", carmaUpdate)
			return carmaUpdate
		}

		// Log failure, continue
		log.Printf(
			"%s: %v: %s: %v", tryStr, errGenFailed, genType, err,
		)
	}

	// Fall back
	log.Println("[model] using fallback value for carma update")
	return carma.Fallback()
}
