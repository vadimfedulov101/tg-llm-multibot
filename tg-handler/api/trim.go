package api

import (
	"strings"
)

// Removes noise
func trimNoise(s string) string {
	s = trimThinkingTags(s)
	s = strings.TrimSpace(s)

	return s
}

// Removes <think>...</think> blocks from string
func trimThinkingTags(s string) string {
	startTag := "<think>"
	endTag := "</think>"

	for {
		startIdx := strings.Index(s, startTag)
		if startIdx == -1 {
			break
		}

		endIdx := strings.Index(s, endTag)
		if endIdx == -1 {
			break
		}

		// Remove the thinking block including tags
		s = s[:startIdx] + s[endIdx+len(endTag):]
	}

	return s
}
