package model

import (
	"strings"
)

// Removes noise
func trimNoise(s string) string {
	s = trimThinkingTags(s)
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "*")
	return s
}

// Smartly removes thinking blocks
func trimThinkingTags(s string) string {
	startTag := "<think>"
	endTag := "</think>"

	// PRIORITY: "Final Answer" comes AFTER the thought process.
	if endIdx := strings.LastIndex(s, endTag); endIdx != -1 {
		return s[endIdx+len(endTag):]
	}

	// FALLBACK: "Final Answer" comes BEFORE the thought process.
	if startIdx := strings.Index(s, startTag); startIdx != -1 {
		return s[:startIdx]
	}

	// No tags found
	return s
}
