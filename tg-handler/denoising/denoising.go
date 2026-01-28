package denoising

import (
	"strings"
)

// Removes noise from model response
func DenoiseResponse(
	s string, botName string, userName string,
) string {

	s = trimThinking(s)
	s = trimNonReply(s, botName, userName)

	return s
}

// Removes thinking part
func trimThinking(s string) string {
	const (
		startTag = "<think>"
		endTag   = "</think>"
	)

	// Skip everything before the last end tag
	if endIdx := strings.LastIndex(s, endTag); endIdx != -1 {
		s = s[endIdx+len(endTag):]
	}
	// Skip everything after the first start tag
	if startIdx := strings.Index(s, startTag); startIdx != -1 {
		s = s[:startIdx]
	}

	s = strings.TrimSpace(s)
	return s
}

// Removes non-reply part
func trimNonReply(s string, botName string, userName string) string {
	var (
		botTag  = botName + ":"
		userTag = userName + ":"
	)

	// Trim bot tag as reply prefix
	s = strings.TrimPrefix(s, botTag)

	// Trim replying for user
	if startIdx := strings.Index(s, userTag); startIdx != -1 {
		s = s[:startIdx]
	}

	s = strings.TrimSpace(s)
	return s
}
