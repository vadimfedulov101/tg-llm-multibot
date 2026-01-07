package tags

import (
	"errors"
	"log"
	"strings"
)

var (
	ErrTagInvalid = errors.New(
		"tag does not start with '#'",
	)
)

// Tag is stored in memory WITHOUT the '#' prefix to save space.
type Tag string

func newTag(s string) (Tag, error) {
	if strings.HasPrefix(s, "#") {
		// Valid tag: strip the '#' and store the rest
		return Tag(s[1:]), nil
	}
	return "", ErrTagInvalid
}

// String adds the '#' back for display purposes (logs, etc.)
func (t Tag) String() string {
	return "#" + string(t)
}

// --- UNIQUE TAGS COLLECTION ---

type UniqueTags []Tag

// NewUniqueTags is for USER INPUT.
// It parses raw string, validates '#' prefix, strips it, dedupes, and applies limit.
func NewUniqueTags(s string, lim int) UniqueTags {
	var tags []Tag

	// Get raw tags slice
	rawTags := strings.Fields(s)

	// Accumulate unique tags
	seen := make(map[Tag]bool)
	for _, rawTag := range rawTags {
		// Try to get tag (validates '#' and strips it)
		tag, err := newTag(rawTag)

		// Skip invalid
		if err != nil {
			log.Println(err)
			continue
		}

		// Skip duplicates
		if seen[tag] {
			continue
		}

		// Add new tag
		seen[tag] = true
		tags = append(tags, tag)

		// Stop if limit reached
		if len(tags) >= lim {
			break
		}
	}

	return tags
}

// DeserializeUniqueTags is for DB/PROTO LOADING.
// It converts a raw space-separated string (no '#') back into UniqueTags.
// It trusts the input and skips validation.
func DeserializeUniqueTags(s string) UniqueTags {
	if s == "" {
		return nil
	}

	// Get raw tags
	rawTags := strings.Fields(s)

	// We cast directly to Tag(rt) because we trust the DB contains clean names
	tags := make(UniqueTags, 0, len(rawTags))
	for _, rt := range rawTags {
		tags = append(tags, Tag(rt))
	}

	return tags
}

// Serialize returns the string WITHOUT brackets and '#' signs.
// Use this for saving to Proto/Database.
func (ts UniqueTags) Serialize() string {
	var sb strings.Builder
	for i, t := range ts {
		if i > 0 {
			sb.WriteString(" ")
		}
		// string(t) bypasses the String() method, giving the raw text ("cool")
		sb.WriteString(string(t))
	}
	return sb.String()
}

// String returns the string WITHOUT brackets but WITH '#' signs.
// Use this for printing to logs or showing the user.
func (ts UniqueTags) String() string {
	var sb strings.Builder
	for i, t := range ts {
		if i > 0 {
			sb.WriteString(" ")
		}
		// t.String() adds the hash back ("#cool")
		sb.WriteString(t.String())
	}
	return sb.String()
}
