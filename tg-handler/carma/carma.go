package carma

import (
	"errors"
	"fmt"
)

const (
	Min = -100
	Max = 100
)

// Carma errors
var (
	errCarmaOOB      = errors.New("carma out of bounds")
	errCarmaBelowMin = fmt.Errorf("below minimum value of %d", Min)
	errCarmaOverMax  = fmt.Errorf("over maximum value of %d", Max)

	errCarmaUpdateOOV = errors.New("carma update out of variants")
)

type Carma int

func New(n int) (*Carma, error) {
	var err = errCarmaOOB

	// Abide bounds
	if n < Min {
		return nil, fmt.Errorf("%w: %v", err, errCarmaBelowMin)
	}
	if n > Max {
		return nil, fmt.Errorf("%w: %v", err, errCarmaOverMax)
	}

	c := Carma(n)
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
		return UpdateNeutral, errCarmaUpdateOOV
	}
}

// Apply carma update
func (c *Carma) Apply(u Update) {
	// Calculate new value
	newVal := int(*c) + int(u)

	// Abide saturation
	if newVal < Min {
		*c = Carma(Min)
		return
	}
	if newVal > Max {
		*c = Carma(Max)
		return
	}

	// Set new value
	*c = Carma(newVal)
}

// Value used in case of generation failure
func Fallback() Update {
	return UpdateNeutral
}
