package model

import (
	"strings"
)

// Cuts hash tags to limit
func cleanTags(tags string, limit int) string {
	// Get raw tags slice
	rawTags := strings.Fields(tags)

	// Set for deduplication
	seen := make(map[string]bool)
	var cleanTags []string

	// Clean tags
	for _, tag := range rawTags {
		// Skip garbage
		if !strings.HasPrefix(tag, "#") {
			continue
		}
		// Check duplicate
		if seen[tag] {
			continue
		}

		// Add clean tag
		seen[tag] = true
		cleanTags = append(cleanTags, tag)

		// Stop if limit reached
		if len(cleanTags) >= limit {
			break
		}
	}

	// Return clean tags
	return strings.Join(cleanTags, " ")
}

// Removes noise
func trimNoise(s string) string {
	s = trimThinkingTags(s)
	s = strings.TrimSpace(s)

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
