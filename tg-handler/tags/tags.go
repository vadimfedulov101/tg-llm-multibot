package tags

import (
	"errors"
	"strings"

	"tg-handler/logging"
)

// Tags errors
var (
	errEmptyRawTagsString = errors.New(
		"empty raw tags string",
	)
	errZeroTags = errors.New(
		"zero tags",
	)
	errTagNoHashSign = errors.New(
		"tag does not start with '#'",
	)
)

// --- PUBLIC TAGS COLLECTION ---

type Tags []tag

// Parses string from LLM and accumulates unique tags from it
func New(s string, lim int, logger *logging.Logger) (Tags, error) {
	// Handle empty string
	if s == "" {
		return nil, errEmptyRawTagsString
	}

	var tags []tag

	// Get raw tags
	rawTags := strings.Fields(s)

	// Accumulate unique tags
	seen := make(map[tag]bool)
	for _, rawTag := range rawTags {
		// Try to get tag
		tag, err := newTag(rawTag)

		// Skip non-tags
		if err != nil {
			logger.Error(
				"failed to create a new tag", logging.Err(err),
			)
			continue
		}

		// Skip duplicates
		if seen[tag] {
			continue
		}

		// Add unique tag
		seen[tag] = true
		tags = append(tags, tag)

		// Stop on limit
		if len(tags) >= lim {
			break
		}
	}

	// Check if non-zero tags
	if len(tags) < 1 {
		return nil, errZeroTags
	}

	return tags, nil
}

// Tags in human-readable format
func (tags Tags) String() string {
	var sb strings.Builder
	for i, tag := range tags {
		if i > 0 {
			sb.WriteString(" ")
		}
		// String() appends '#' prefix
		sb.WriteString(tag.String())
	}
	return sb.String()
}

// Tags in machine-readable fromat
func (tags Tags) Serialize() string {
	var sb strings.Builder
	for i, tag := range tags {
		if i > 0 {
			sb.WriteString(" ")
		}
		// string() bypasses String() method not adding '#' prefix
		sb.WriteString(string(tag))
	}
	return sb.String()
}

// Tags from machine-readable format
func DeserializeTags(s string) Tags {
	if s == "" {
		return nil
	}

	// Get raw tags
	rawTags := strings.Fields(s)

	// Accumulate tags casted from raw tags
	tags := make(Tags, 0, len(rawTags))
	for _, rawTag := range rawTags {
		// Cast directly with trust in type check
		tag := tag(rawTag)
		// Append to tags
		tags = append(tags, tag)
	}

	return tags
}

func Fallback() Tags {
	return Tags{"unknown"}
}

// --- PRIVATE TAGS TYPE ---

type tag string

// Tag starts with '#': dropped for memory, implied for printing
func newTag(s string) (tag, error) {
	if strings.HasPrefix(s, "#") {
		return tag(s[1:]), nil
	}
	return "", errTagNoHashSign
}

// Tag in human-readable format
func (t tag) String() string {
	return "#" + string(t)
}
