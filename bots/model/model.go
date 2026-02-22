package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"telellama/conf"
	"telellama/denoising"
	"telellama/karma"
	"telellama/logging"
	"telellama/memory"
	"telellama/names"
	"telellama/prompts"
	"telellama/selectIdx"
	"telellama/tags"
)

// Constants
const (
	envModelVar  = "OLLAMA_MODEL"
	envApiUrlVar = "OLLAMA_API_URL"
	defApiUrl    = "http://ollama:11434/api/generate"
	retryTime    = 10 * time.Second
	waitTimeout  = 5 * time.Minute
	maxSelectTry = 5
	maxTagsTry   = 5
	maxKarmaTry  = 5
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
	errGetEnvFailed = errors.New("failed to get env variable")
	errGenFailed    = errors.New("generation failed")
)

// LLM model
type Model struct {
	Name      string
	ApiUrl    string
	Config    *conf.BotConf
	Prompts   *prompts.Prompts
	Memory    *memory.Memory
	Names     *names.Names
	ChatTitle string
	Logger    *logging.Logger
}

func New(
	botConf *conf.BotConf,
	prompts *prompts.Prompts,
	memory *memory.Memory,
	names *names.Names,
	chatTitle string,
	logger *logging.Logger,
) *Model {
	const errMsg = "failed to get env variable"

	// Get model name
	name, ok := os.LookupEnv(envModelVar)
	if !ok {
		logger.With(
			logging.EnvVar(envModelVar),
		).Panic(errMsg, logging.Err(errGetEnvFailed))
	}

	// Get API URL (default to internal docker DNS if not set)
	apiUrl, ok := os.LookupEnv(envApiUrlVar)
	if !ok {
		apiUrl = defApiUrl
	}

	return &Model{
		Name:      name,
		ApiUrl:    apiUrl,
		Config:    botConf,
		Prompts:   prompts,
		Memory:    memory,
		Names:     names,
		ChatTitle: chatTitle,
		Logger:    logger,
	}
}

// Replies to new message as model
func (m *Model) Reply(ctx context.Context) (string, error) {
	candidates, err := m.genCandidates(ctx)
	if errors.Is(err, ErrCtxDone) {
		return "", err
	}

	bestCandidate, err := m.selectBestCandidate(ctx, candidates)
	if errors.Is(err, ErrCtxDone) {
		return "", err
	}

	return bestCandidate, nil
}

// Reflects on response
func (m *Model) Reflect(ctx context.Context, user string) error {
	var (
		botContacts = m.Memory.BotContacts
	)

	// Get contact to update
	botContact := botContacts.Get(user)

	// Update karma
	karmaUpdate, err := m.genKarmaUpdate(ctx)
	if errors.Is(err, ErrCtxDone) {
		return err
	}
	botContact.Karma.Apply(karmaUpdate)

	// Update persona
	tags, err := m.genTags(ctx)
	if errors.Is(err, ErrCtxDone) {
		return err
	}
	botContact.Tags = tags

	// Reset contacts
	botContacts.Set(user, botContact)

	return nil
}

// Generates candidates
func (m *Model) genCandidates(
	ctx context.Context,
) ([]string, error) {
	logger := m.Logger

	var (
		candidateNum = m.Config.Main.CandidateNum
		candidates   = make([]string, 0, candidateNum)
	)

	// Get start time
	start := time.Now()

	// Form request
	request := m.newRequest(m.Prompts.Response)

	// Generate candidates
	for i := range candidateNum {
		// Get iteration start time
		iStart := time.Now()

		// Log start
		iterLog := logger.With(logging.Iter(i + 1))
		iterLog.Info("generating candidate")

		// Get new candidate
		candidate, err := sendRequestEternal(ctx, request, iterLog)
		if errors.Is(err, ErrCtxDone) {
			return []string{}, ErrCtxDone
		}

		// Append to candidates
		candidates = append(candidates, candidate)

		// Log successs
		iterLog.Info(
			"candidate generated",
			logging.Candidate(candidate),
		)
		iterLog.Debug(
			"candidate generation took",
			logging.Duration(time.Since(iStart)),
		)
	}

	// Log final success
	logger.Info("candidates generated")
	logger.Debug(
		"canidates generation took",
		logging.Duration(time.Since(start)),
	)
	return candidates, nil
}

// Select the best candidate
func (m *Model) selectBestCandidate(
	ctx context.Context,
	candidates Candidates,
) (string, error) {
	logger := m.Logger

	// One candidate to be selected from, return it
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	// Get start time
	start := time.Now()

	// Format prompt
	prompt := prompts.FinFmtSelectPrompt(
		m.Prompts.Select, candidates,
	)
	// Form request
	request := m.newRequest(prompt)

	// Try to select the best candidate
	for i := range maxSelectTry {
		// Log start
		iterLog := logger.With(logging.Iter(i + 1))
		iterLog.Info("selecting candidate")

		// Try to get select index
		selectStr, err := sendRequestEternal(ctx, request, iterLog)
		if errors.Is(err, ErrCtxDone) {
			return "", err
		}
		selectIdx, err := selectIdx.New(selectStr, len(candidates))

		// Log success, return
		if err == nil {
			candidateSelected := candidates[selectIdx]
			iterLog.Info(
				"candidate selected",
				logging.Candidate(candidateSelected),
			)
			iterLog.Debug(
				"candidate selection took",
				logging.Duration(time.Since(start)),
			)
			return candidateSelected, nil
		}

		// Log failure, continue
		iterLog.Error("selection failed", logging.Err(
			fmt.Errorf("%w: %v", errGenFailed, err),
		))
	}

	// Fall back
	logger.Error("using fallback value for candidates")
	return candidates.Fallback(), nil
}

// Generates unique tags
func (m *Model) genTags(
	ctx context.Context,
) (tags.Tags, error) {
	logger := m.Logger

	// Get start time
	start := time.Now()

	// Form request
	request := m.newRequest(m.Prompts.Tags)

	for i := range maxTagsTry {
		// Log start
		iterLog := logger.With(logging.Iter(i + 1))
		iterLog.Info("generating tags")

		// Get tags
		rawTags, err := sendRequestEternal(ctx, request, iterLog)
		if errors.Is(err, ErrCtxDone) {
			return nil, err
		}
		tags, err := tags.New(rawTags, m.Memory.Limits.Tags, iterLog)

		// Log success, return
		if err == nil {
			iterLog.Info(
				"tags generated",
				logging.Tags(tags.String()),
			)
			iterLog.Debug(
				"tags generation took",
				logging.Duration(time.Since(start)),
			)
			return tags, nil
		}

		// Log failure, continue
		iterLog.Error(
			"generating tags failed", logging.Err(
				fmt.Errorf("%w: %v", errGenFailed, err),
			),
		)
	}

	// Fall back
	logger.Error("using fallback value for tags")
	return tags.Fallback(), nil
}

// Generates karma update
func (m *Model) genKarmaUpdate(
	ctx context.Context,
) (karma.Update, error) {
	logger := m.Logger

	// Get start time
	start := time.Now()

	// Form request
	request := m.newRequest(m.Prompts.Karma)

	for i := range maxKarmaTry {
		// Log start
		iterLog := logger.With(logging.Iter(i + 1))
		iterLog.Info("generating karma update")

		// Try to get karma update
		karmaUpdateStr, err := sendRequestEternal(ctx, request, iterLog)
		if errors.Is(err, ErrCtxDone) {
			return karma.Fallback(), err
		}
		karmaUpdate, err := karma.NewUpdate(karmaUpdateStr)

		// Log success, return
		if err == nil {
			iterLog.Info(
				"karma update generated",
				logging.KarmaUpdate(karmaUpdate.String()),
			)
			iterLog.Debug(
				"karma update generation took",
				logging.Duration(time.Since(start)),
			)
			return karmaUpdate, nil
		}

		// Log failure, continue
		iterLog.Error(
			"failed to generate karma update",
			logging.Err(
				fmt.Errorf("%w: %v", errGenFailed, err),
			),
		)
	}

	// Fall back
	logger.Error("using fallback value for karma update")
	return karma.Fallback(), nil
}

// Forms new request using model's model and config
func (m *Model) newRequest(prompt string) *Request {
	return newRequest(
		prompt, m.ApiUrl, m.Name, m.Config, m.getReplyCleaner(),
	)
}

// Gets reply cleaner
func (m *Model) getReplyCleaner() func(string) string {
	var names = m.Names
	var (
		botName  = names.Bot
		userName = names.User
	)

	return func(text string) string {
		return denoising.DenoiseResponse(text, botName, userName)
	}
}
