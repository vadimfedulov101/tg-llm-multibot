package model

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"tg-handler/conf"
	"tg-handler/memory"
	"tg-handler/prompts"
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

// API constants
const (
	model       = "huihui_ai/qwen3-abliterated:14b-q8_0"
	apiUrl      = "http://ollama:11434/api/generate"
	retryTime   = time.Minute
	waitTimeout = 10 * time.Minute
)

// Retry constants
const (
	maxSelectTry = 10
	maxTagsTry   = 10
	maxCarmaTry  = 10
)

// Inference errors
var (
	// SELECT
	// General error
	ErrSelectFailed = errors.New("[model] selection failed")
	// Specific errors to be wrapped
	ErrNoNum = errors.New(
		"no numbers found in response",
	)
	ErrIdxNaN = errors.New(
		"candidate index is not a number",
	)
	ErrIdxOOB = errors.New(
		"candidate index is out of bounds",
	)
	ErrInvalidCandidateNum = errors.New(
		"negative or zero candidate num",
	)

	// Carma
	// General error
	ErrCarmaUpdateFailed = errors.New("[model] carma update failed")
	// Specific errors to be wrapped
	ErrEnumOOV = errors.New("carma update outside of enum variants")
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
		botConf, promptTemplates,
		memory, lastMsg, botName, chatTitle,
	)

	// Return model via pointer
	return &Model{
		Config:    botConf,
		Prompts:   prompts,
		Memory:    memory,
		BotName:   botName,
		ChatTitle: chatTitle,
	}
}

// Candidates representation
type Candidates []string

func (cs Candidates) String() (s string) {
	var sb strings.Builder
	for i, candidate := range cs {
		sb.WriteString(
			fmt.Sprintf("%d) %s\n\n", i+1, candidate),
		)
	}
	return sb.String()
}

// Reacts to new message
func (m *Model) React(ctx context.Context) string {
	// Generate candidate(s)
	candidates := m.generateCandidates(ctx)
	if m.Config.Main.CandidateNum == 1 {
		return candidates[0]
	}

	// Select candidate
	candidateIdx := m.selectCandidate(ctx, candidates)

	return candidates[candidateIdx]
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

	// Update bot contact carma
	carmaUpdate := m.updateCarma(ctx, msg.Line())
	botContact.Carma += int(carmaUpdate)

	// Update bot contact persona
	tags := m.updateTags(ctx, msg.Line())
	botContact.Tags = tags

	// Update bot contacts with new bot contact
	botContacts.Set(sender, botContact)
}

// Generates candidates
func (m *Model) generateCandidates(ctx context.Context) []string {
	var (
		candidateNum = m.Config.Main.CandidateNum
		candidates   = make([]string, 0, candidateNum)
	)
	if candidateNum <= 0 {
		log.Fatalf("%v: %v", ErrSelectFailed, ErrInvalidCandidateNum)
	}

	// Create request
	request := newRequest(m.Prompts.Response, m.Config)

	// Generate candidates
	for i := range candidateNum {
		// Get new candidate
		candidate := sendRequestEternal(ctx, request)

		// Add new candidate
		candidates = append(candidates, candidate)

		log.Printf("[model] candidate %d: %s", i+1, candidate)
	}

	return candidates
}

// Selects candidate from all candidates
func (m *Model) selectCandidate(
	ctx context.Context,
	candidates Candidates,
) int {
	re := regexp.MustCompile(`\b(\d+)\b`)

	// Finalizes prompt formatting
	prompt := prompts.FinFmtSelectPrompt(m.Prompts.Select, candidates)

	// Create request
	request := newRequest(prompt, m.Config)

	// Try to generate selection index
	var err error
	for i := range maxSelectTry {
		// Log try and error if not the first try
		if i == 0 {
			log.Printf("Select try: %d", i+1)
		} else if i > 0 {
			log.Printf(
				"Select try %d: %v: %v",
				i+1, ErrSelectFailed, err,
			)
		}

		// Send request
		selectText := sendRequestEternal(ctx, request)

		// Get last number in select text
		matches := re.FindAllString(selectText, -1)
		if len(matches) == 0 {
			err = ErrNoNum
			continue // Fail
		}
		lastNumStr := matches[len(matches)-1]

		// Get selection number
		selectNum, convErr := strconv.Atoi(lastNumStr)
		if err != nil {
			err = fmt.Errorf("%w: %v", ErrIdxNaN, convErr)
			continue // Fail
		}

		// Validate index
		candidateIdx := selectNum - 1
		if candidateIdx >= 0 && candidateIdx < len(candidates) {
			log.Printf("[model] candidate selected: %d", selectNum)
			return candidateIdx // Success
		}
		err = fmt.Errorf("%w: %d not in (0-%d)",
			ErrIdxOOB, candidateIdx, len(candidates),
		)
	}

	// Select 1-st candidate on fail
	log.Printf("[model] candidate selected: %d", 1)
	return 0
}

// Generates new tags
func (m *Model) updateTags(
	ctx context.Context,
	line string,
) string {
	// Finalize tags prompt formatting
	prompt := prompts.FinFmtTagsPrompt(m.Prompts.Tags, line)

	// Create request
	request := newRequest(prompt, m.Config)

	// Get new tags
	tags := sendRequestEternal(ctx, request)

	// Clean tags
	tags = cleanTags(tags, m.Memory.Limits.Tags)

	log.Printf("[model] tags: %s", tags)
	return tags
}

// Generates carma update
func (m *Model) updateCarma(
	ctx context.Context,
	line string,
) CarmaUpdate {
	// Finalize carma prompt formatting
	prompt := prompts.FinFmtCarmaPrompt(m.Prompts.Carma, line)

	// Create request
	request := newRequest(prompt, m.Config)

	var err error
	for i := range maxCarmaTry {
		// Log previous try error
		if i > 0 {
			log.Printf(
				"Carma try %d: %v: %v",
				i+1, ErrCarmaUpdateFailed, err,
			)
		}

		// Send request
		carmaUpdateS := sendRequestEternal(ctx, request)

		// Get carma update
		carmaUpdate, err := NewCarmaUpdate(carmaUpdateS)
		if err == nil {
			log.Printf("[model] carma update: %s", carmaUpdate)
			return carmaUpdate
		}
	}

	// Return no carma update on fail
	log.Printf("[model] carma update: %s", CarmaUpdateNeutral)
	return CarmaUpdateNeutral
}
