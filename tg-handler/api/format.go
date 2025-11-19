package api

import (
	"fmt"
	"strings"

	"tg-handler/memory"
)

// Formats prompt with memory
func fmtMemory(prompt string, memory *memory.Memory) string {
	chatContext := strings.Join(memory.ChatContext, "\n")
	replyChain := strings.Join(memory.ReplyChain, "\n")
	return fmt.Sprintf(prompt, chatContext, replyChain)
}

// Formats prompt with candidates
func fmtCandidates(prompt string, candidates []string) string {
	var candidate_s string
	for i, candidate := range candidates {
		candidate_s += fmt.Sprintf("[%d] ", i+1) + candidate + "\n\n"
	}
	return fmt.Sprintf(prompt, candidate_s)
}

// trimThinkingTags removes thinking tags from the response
func trimThinkingTags(response string) string {
	// Remove <think>...</think> blocks
	startTag := "<think>"
	endTag := "</think>"

	for {
		startIdx := strings.Index(response, startTag)
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(response, endTag)
		if endIdx == -1 {
			break
		}

		// Remove the thinking block including tags
		response = response[:startIdx] + response[endIdx+len(endTag):]
	}

	response = strings.Trim(response, "\n")
	response = strings.TrimSpace(response)

	return response
}
