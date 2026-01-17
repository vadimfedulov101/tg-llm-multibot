package selectIdx

import (
	"errors"
	"fmt"
	"strconv"
)

// Selection errors
var (
	ErrSelectNumNaN = errors.New(
		"select number is not a number",
	)
	ErrSelectIdxOOB = errors.New(
		"select index is out of bounds",
	)
)

type SelectIdx int

func New(s string, lim int) (SelectIdx, error) {
	// Convert number
	num, err := strconv.Atoi(s)
	if err != nil {
		return 0, ErrSelectNumNaN
	}

	// Calculate index
	idx := num - 1

	// Check index bounds
	if idx < 0 || idx > lim {
		return 0, ErrSelectIdxOOB
	}

	return SelectIdx(num), nil
}

// Select index in human-readable format
func (si SelectIdx) String() string {
	return fmt.Sprintf("selection index %d", si)
}
