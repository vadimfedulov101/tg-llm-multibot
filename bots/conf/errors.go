package conf

import (
	"errors"
)

// Config errors
var (
	// Content errors
	errEmptyTemplate = errors.New("empty template for config")

	// I/O errors
	errReadFailed      = errors.New("read config failed")
	errUnmarshalFailed = errors.New("unmarshal config failed")

	// Placeholder errors
	errWrongPlaceholderNum  = errors.New("wrong placeholder number")
	errPlaceholderOverflow  = errors.New("counted more than needed")
	errPlaceholderUnderflow = errors.New("counted less than needed")

	// Bot config errors
	errNegCandidateNum = errors.New("negative candidate number")
)
