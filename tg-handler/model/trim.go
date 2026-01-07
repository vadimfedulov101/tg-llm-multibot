package model

import (
	"strings"
)

// Cuts tags to limit
func cutTags(s string, lim int) string {
	// Get raw tags slice
	oldTags := strings.Fields(s)

	// Set for deduplication
	seen := make(map[string]bool)
	var newTags []string

	// Cut tags with cleaning and deduplication
	for _, tag := range oldTags {
		// Skip not-hashtags
		if !strings.HasPrefix(tag, "#") {
			continue
		}
		// Skip duplicates
		if seen[tag] {
			continue
		}

		// Add new tag
		seen[tag] = true
		newTags = append(newTags, tag)

		// Stop if limit reached
		if len(newTags) >= lim {
			break
		}
	}

	// Return new tags
	return strings.Join(newTags, " ")
}

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
