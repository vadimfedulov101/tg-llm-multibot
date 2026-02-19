package karma

import (
	"errors"
	"fmt"
)

const (
	Min = -100
	Max = 100
)

// Karma errors
var (
	errKarmaOOB      = errors.New("karma out of bounds")
	errKarmaBelowMin = fmt.Errorf("below minimum value of %d", Min)
	errKarmaOverMax  = fmt.Errorf("over maximum value of %d", Max)

	errKarmaUpdateOOV = errors.New("karma update out of variants")
)

type Karma int

func New(n int) (*Karma, error) {
	var err = errKarmaOOB

	// Abide bounds
	if n < Min {
		return nil, fmt.Errorf("%w: %v", err, errKarmaBelowMin)
	}
	if n > Max {
		return nil, fmt.Errorf("%w: %v", err, errKarmaOverMax)
	}

	c := Karma(n)
	return &c, nil
}

type Update int

const UpdateDelta = 10
const (
	UpdateNegative Update = -UpdateDelta
	UpdateNeutral         = 0
	UpdatePositive        = UpdateDelta
)

// To string
var UpdateTag = map[Update]string{
	UpdateNegative: "-",
	UpdateNeutral:  "=",
	UpdatePositive: "+",
}

func (u Update) String() string {
	return UpdateTag[u]
}

// From string
func NewUpdate(s string) (Update, error) {
	switch s {
	case "-":
		return UpdateNegative, nil
	case "=":
		return UpdateNeutral, nil
	case "+":
		return UpdatePositive, nil
	default:
		return UpdateNeutral, errKarmaUpdateOOV
	}
}

// Apply karma update
func (c *Karma) Apply(u Update) {
	// Calculate new value
	newVal := int(*c) + int(u)

	// Abide saturation
	if newVal < Min {
		*c = Karma(Min)
		return
	}
	if newVal > Max {
		*c = Karma(Max)
		return
	}

	// Set new value
	*c = Karma(newVal)
}

// Value used in case of generation failure
func Fallback() Update {
	return UpdateNeutral
}
