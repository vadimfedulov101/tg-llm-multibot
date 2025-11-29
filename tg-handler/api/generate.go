package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"time"

	"tg-handler/conf"
	"tg-handler/history"
)

// Ollama constants
const (
	model        = "huihui_ai/qwen3-abliterated:14b-q8_0"
	apiUrl       = "http://ollama:11434/api/generate"
	maxSelectTry = 10
	retryTime    = time.Minute
	waitTimeout  = 10 * time.Minute
)

// Generation errors
var (
	// General error
	ErrSelect = errors.New("[api] selection failed")
	// Specific errors to be wrapped
	ErrIdxNaN         = errors.New("candidate index is not a number")
	ErrIdxOutOfBounds = errors.New("candidate index is out of bounds")
)

// Generates response
func Generate(
	ctx context.Context,
	memory *history.Memory,
	confPath string,
	prompts *conf.Prompts,
	botName string,
	chatTitle string,
) string {
	// Load settings from config (chat title in system prompt)
	settings := mustLoadSettings(confPath)
	settings.BotConf.SystemPrompt = fmtSystemPrompt(
		settings.BotConf.SystemPrompt, chatTitle,
	)

	// Generate candidates
	candidates := generateCandidates(ctx, settings, memory, prompts, botName)
	if settings.BotConf.CandidateNum == 1 {
		return candidates[0]
	}

	// Select best candidate
	candidateIdx := selectBestCandidate(
		ctx, settings, memory, prompts, candidates,
	)

	return candidates[candidateIdx]
}

// Generates <candidateNum> candidates
func generateCandidates(
	ctx context.Context,
	settings *Settings,
	memory *history.Memory,
	prompts *conf.Prompts,
	botName string,
) []string {
	// Load candidate num
	candidateNum := settings.BotConf.CandidateNum

	// Initialize candidates
	candidates := make([]string, 0, candidateNum)

	// Create request
	prompt := fmtResponsePrompt(prompts.Response, memory, botName)
	request := newRequest(prompt, settings)

	// Generate selection candidates
	for i := range candidateNum {
		// Get new candidate via request
		candidate := sendRequestEternal(ctx, request)

		// Add new candidate
		candidates = append(candidates, trimNoise(candidate))

		log.Printf("[api] candidate %d: %s", i+1, candidate)
	}

	return candidates
}

// Selects the best candidate from <len(candidates)> candidates
func selectBestCandidate(
	ctx context.Context,
	settings *Settings,
	memory *history.Memory,
	prompts *conf.Prompts,
	candidates []string,
) (candidateIdx int) {
	// Load candidate num
	candidateNum := settings.BotConf.CandidateNum

	// Create request
	prompt := fmtSelectPrompt(prompts.Select, memory, candidates, candidateNum)
	request := newRequest(prompt, settings)

	// Try to generate selection index
	var err error
	for i := range maxSelectTry {
		// Log previous try error
		if i > 0 {
			log.Printf("Select try %d: %v", i+1, err)
		}

		// Send request
		selectText := sendRequestEternal(ctx, request)

		// Get selection number
		selectNum, err := strconv.Atoi(trimNoise(selectText))
		if err != nil {
			err = fmt.Errorf("%w: %v", ErrIdxNaN, err)
			continue // Fail
		}

		// Validate index
		candidateIdx := selectNum - 1
		if candidateIdx > 0 && candidateIdx < len(candidates) {
			return candidateIdx // Success
		}
		err = fmt.Errorf("%w: %d not in (0-%d)",
			ErrIdxOutOfBounds, candidateIdx, len(candidates),
		)
	}

	// Select random candidate index (as generation failed)
	candidateIdx = rand.IntN(len(candidates))

	return candidateIdx
}
