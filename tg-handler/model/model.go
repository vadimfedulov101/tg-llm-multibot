package model

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"tg-handler/carma"
	"tg-handler/conf"
	"tg-handler/denoising"
	"tg-handler/logging"
	"tg-handler/memory"
	"tg-handler/names"
	"tg-handler/prompts"
	"tg-handler/selectIdx"
	"tg-handler/tags"
)

// Constants
const (
	envModelVar  = "LLM_MODEL"
	apiUrl       = "http://ollama:11434/api/generate"
	retryTime    = 10 * time.Second
	waitTimeout  = 2 * time.Minute
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
	errGetEnvFailed = errors.New("failed to get env variable")
	errGenFailed    = errors.New("generation failed")
)

// LLM model
type Model struct {
	Name      string
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
		logger.With(logging.EnvVar(envModelVar)).
			Panic(errMsg, logging.Err(errGetEnvFailed))
	}

	return &Model{
		Name:      name,
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
func (m *Model) Reflect(
	ctx context.Context,
	user string,
	reply Message,
) error {
	var (
		botContacts = m.Memory.BotContacts
	)

	// Get contact to update
	botContact := botContacts.Get(user)

	// Update carma
	carmaUpdate, err := m.genCarmaUpdate(ctx, reply.Line())
	if errors.Is(err, ErrCtxDone) {
		return err
	}
	botContact.Carma.Apply(carmaUpdate)

	// Update persona
	tags, err := m.genTags(ctx, reply.Line())
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
		iterLog.Debug(
			"candidate generated",
			logging.Candidate(candidate),
			logging.Duration(time.Since(iStart)),
		)
	}

	// Log final success
	logger.With(
		logging.Duration(time.Since(start)),
	).Info("candidates generated")
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
	logger.Info("using fallback value for candidates")
	return candidates.Fallback(), nil
}

// Generates unique tags
func (m *Model) genTags(
	ctx context.Context,
	replyLine string,
) (tags.Tags, error) {
	logger := m.Logger

	// Get start time
	start := time.Now()

	// Format prompt
	prompt := prompts.FinFmtTagsPrompt(m.Prompts.Tags, replyLine)
	// Form request
	request := m.newRequest(prompt)

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
	logger.Info("using fallback value for tags")
	return tags.Fallback(), nil
}

// Generates carma update
func (m *Model) genCarmaUpdate(
	ctx context.Context,
	replyLine string,
) (carma.Update, error) {
	logger := m.Logger

	// Get start time
	start := time.Now()

	// Format prompt
	prompt := prompts.FinFmtCarmaPrompt(m.Prompts.Carma, replyLine)
	// Form request
	request := m.newRequest(prompt)

	for i := range maxCarmaTry {
		// Log start
		iterLog := logger.With(logging.Iter(i + 1))
		iterLog.Info("generating carma update")

		// Try to get carma update
		carmaUpdateStr, err := sendRequestEternal(ctx, request, iterLog)
		if errors.Is(err, ErrCtxDone) {
			return carma.Fallback(), err
		}
		carmaUpdate, err := carma.NewUpdate(carmaUpdateStr)

		// Log success, return
		if err == nil {
			iterLog.Info(
				"carma update generated",
				logging.CarmaUpdate(carmaUpdate.String()),
				logging.Duration(time.Since(start)),
			)
			return carmaUpdate, nil
		}

		// Log failure, continue
		iterLog.Error(
			"failed to generate carma update",
			logging.Err(
				fmt.Errorf("%w: %v", errGenFailed, err),
			),
		)
	}

	// Fall back
	logger.Info("using fallback value for carma update")
	return carma.Fallback(), nil
}

// Forms new request using model's model and config
func (m *Model) newRequest(prompt string) *Request {
	return newRequest(prompt, m.Name, m.Config, m.getReplyCleaner())
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
