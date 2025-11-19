package api

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"tg-handler/initconf"
	"tg-handler/memory"
)

// Ollama constants
const (
	OLLAMA_MODEL   = "huihui_ai/qwen3-abliterated:14b-q8_0"
	OLLAMA_API     = "http://ollama:11434/api/generate"
	MAX_SEND_TRY   = 3
	MAX_SELECT_TRY = 10
	RETRY_TIME     = 5 * time.Second
	API_TIMEOUT    = 10 * time.Minute
)

// Generates response
func Generate(
	ctx context.Context,
	conf string,
	chatTitle string,
	mem *memory.Memory,
	prompts *initconf.Prompts,
	candidateNum int,
) (string, error) {
	// Load settings from config (with chat title in system prompt)
	settings, err := loadSettings(conf, chatTitle)
	if err != nil {
		return "", fmt.Errorf("Failed to load settings: %w", err)
	}

	// Generate candidates
	candidates, err := generateCandidates(
		ctx, settings, mem, prompts, candidateNum,
	)
	if err != nil {
		return "", fmt.Errorf("Failed to generate candidates: %w", err)
	}

	// Select best candidate
	candidateIdx, err := selectBestCandidate(
		ctx, settings, mem, prompts, candidates,
	)
	if err != nil {
		return "", fmt.Errorf("Failed to select candidate: %w", err)
	}

	// Return best candidate
	return candidates[candidateIdx], nil
}

// Generates <candidateNum> candidates
func generateCandidates(
	ctx context.Context,
	settings *Settings,
	mem *memory.Memory,
	prompts *initconf.Prompts,
	candidateNum int,
) ([]string, error) {
	// Initialize candidates
	candidates := make([]string, 0, candidateNum)

	// Create request
	prompt := fmtMemory(prompts.ResponsePrompt, mem)
	request := newOllamaRequest(prompt, settings)

	// Generate selection candidates
	for i := range candidateNum {
		// Send request
		candidate, err := sendRequestExhaustive(ctx, request)
		if err != nil {
			return candidates, fmt.Errorf(
				"All %d send tries exhausted. Last error: %w", MAX_SEND_TRY, err,
			)

		}
		candidate = trimThinkingTags(candidate)
		candidates = append(candidates, candidate)
		log.Printf("Candidate %d: %s", i+1, candidate)
	}

	return candidates, nil
}

// Selects the best candidate from <candidateNum> candidates
func selectBestCandidate(
	ctx context.Context,
	settings *Settings,
	mem *memory.Memory,
	prompts *initconf.Prompts,
	candidates []string,
) (candidateIdx int, err error) {
	// Create request
	prompt := fmtCandidates(fmtMemory(prompts.SelectPrompt, mem), candidates)
	request := newOllamaRequest(prompt, settings)

	// Generate selection index
	for i := range MAX_SELECT_TRY {
		// Send request
		text, err := sendRequestExhaustive(ctx, request)
		if err != nil {
			return 0, fmt.Errorf(
				"All %d send tries exhausted. Last error: %w", MAX_SEND_TRY, err,
			)
		}

		// Try to convert text
		candidateSelection, err := strconv.Atoi(trimThinkingTags(text))
		candidateIdx := candidateSelection - 1
		isIdxValid := candidateIdx > 0 && candidateIdx < len(candidates)
		if err == nil && isIdxValid { // Exit on success: got valid idx in bounds
			break
		}
		// Log specific error
		if !isIdxValid {
			err = fmt.Errorf(
				"Candidate index %d is out of bounds (0-%d): %s",
				candidateIdx, len(candidates), text)
		} else {
			err = fmt.Errorf("Candidate index is NaN: %w", err)
		}
		log.Printf("Select try %d: %v", i+1, err)
	}

	return candidateIdx, nil
}
