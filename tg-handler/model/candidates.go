package model

import (
	"fmt"
	"strings"
)

type Candidates []string

func (cs Candidates) String() (s string) {
	var sb strings.Builder

	for i, candidate := range cs {
		sb.WriteString(
			fmt.Sprintf("%d) %s\n\n", i+1, candidate),
		)
	}

	return sb.String()
}
