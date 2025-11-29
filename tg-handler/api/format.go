package api

import (
	"fmt"
	"log"

	"tg-handler/history"
)

// Formats system prompt with chat title
func fmtSystemPrompt(systemPrompt string, chatTitle string) string {
	return fmt.Sprintf(systemPrompt, chatTitle)
}

// Formats response prompt
func fmtResponsePrompt(prompt string, memory *history.Memory, botName string) string {
	// Get strings
	shortMemoryS, longMemoryS := history.GetMemoryStrings(memory)

	// Format
	responsePrompt := fmt.Sprintf(prompt, shortMemoryS, longMemoryS, botName)

	log.Printf("[api] response prompt: %s", responsePrompt)
	return responsePrompt
}

// Formats select prompt
func fmtSelectPrompt(
	prompt string,
	memory *history.Memory,
	candidates []string,
	candidateNum int,
) string {
	// Get strings
	var (
		shortMemoryS, longMemoryS = history.GetMemoryStrings(memory)
		candidatesS               = getCandidatesString(candidates)
	)

	// Format
	selectPrompt := fmt.Sprintf(
		prompt, shortMemoryS, longMemoryS, candidatesS, candidateNum,
	)

	log.Printf("[api] select prompt: %s", selectPrompt)
	return selectPrompt
}

// Formats prompt with candidates
func getCandidatesString(candidates []string) (s string) {
	for i, candidate := range candidates {
		s += fmt.Sprintf("[%d] ", i+1) + candidate + "\n\n"
	}
	return s
}
