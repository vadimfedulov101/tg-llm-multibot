package carma

import (
	"errors"
	"fmt"
)

const (
	Min = 0
	Max = 100
)

// Carma errors
var (
	ErrCarmaOOB = errors.New("[carma] out of bounds")

	ErrCarmaBelowMin = errors.New("carma below minimum value")
	ErrCarmaOverMax  = errors.New("carma over maximum value")

	ErrCarmaUpdateOOV = errors.New("carma update out of variants")
)

type Carma int

func New(n int) (Carma, error) {
	// Check bounds
	var err = ErrCarmaOOB
	if n < Min {
		return Carma(n), fmt.Errorf("%w: %v", err, ErrCarmaBelowMin)
	}
	if n > Max {
		return Carma(n), fmt.Errorf("%w: %v", err, ErrCarmaOverMax)
	}

	return Carma(n), nil
}

type Update int

const UpdateDelta = 10
const (
	UpdateNegative Update = -UpdateDelta
	UpdateNeutral         = 0
	UpdatePositive        = UpdateDelta
)

// To string
var UpdateName = map[Update]string{
	UpdateNegative: "-",
	UpdateNeutral:  "=",
	UpdatePositive: "+",
}

func (u Update) String() string {
	return UpdateName[u]
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
		return UpdateNeutral, ErrCarmaUpdateOOV
	}
}

// Apply update to carma
func (c *Carma) Apply(u Update) {
	// Calculate new value
	newVal := int(*c) + int(u)

	// Limit to minimum
	if newVal < Min {
		*c = Carma(Min)
		return
	}

	// Limit to maximum
	if newVal > Max {
		*c = Carma(Max)
		return
	}

	// 4. Set valid value
	*c = Carma(newVal)
}
