package model

import (
	"errors"
	"strconv"
)

// Selection errors
var (
	ErrSelectOptNaN = errors.New(
		"selection option is not a number",
	)
	ErrSelectIdxOOB = errors.New(
		"selection index is out of bounds",
	)
)

type SelectIdx int

func newSelectIdx(s string, lim int) (SelectIdx, error) {
	// Convert option to number
	num, err := strconv.Atoi(s)
	if err != nil {
		return 0, ErrSelectOptNaN
	}

	// Calculate index
	idx := num - 1

	// Check index bounds
	if idx < 0 || idx > lim {
		return 0, ErrSelectIdxOOB
	}

	return SelectIdx(num), nil
}
