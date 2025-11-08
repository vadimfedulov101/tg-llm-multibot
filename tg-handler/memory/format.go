package memory

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Formats message fields into line
func Format(message IMessage) string {
	var result string

	text := message.GetText()
	sender := message.GetSender()
	order := message.GetOrder()

	// Handle empty fields
	if text == "" || sender == "" {
		return ""
	}

	// Format
	if order != "" {
		// "text" (no order)
		if strings.HasSuffix(text, order) {
			result = strings.TrimSuffix(text, order)
		}
	} else {
		// "Name: text"
		titleizer := cases.Title(language.English)
		senderTitleized := titleizer.String(sender)

		result = senderTitleized + ": " + text
	}

	return result
}
