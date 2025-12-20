package model

type CarmaUpdate int

const CarmaUpdateDelta = 10
const (
	CarmaUpdateNegative CarmaUpdate = -CarmaUpdateDelta
	CarmaUpdateNeutral              = 0
	CarmaUpdatePositive             = CarmaUpdateDelta
)

// To string
var CarmaUpdateName = map[CarmaUpdate]string{
	CarmaUpdateNegative: "-",
	CarmaUpdateNeutral:  "=",
	CarmaUpdatePositive: "+",
}

func (cu CarmaUpdate) String() string {
	return CarmaUpdateName[cu]
}

// From string
func NewCarmaUpdate(s string) (CarmaUpdate, error) {
	switch s {
	case "-":
		return CarmaUpdateNegative, nil
	case "=":
		return CarmaUpdateNeutral, nil
	case "+":
		return CarmaUpdatePositive, nil
	default:
		return CarmaUpdateNeutral, ErrEnumOOV
	}
}
