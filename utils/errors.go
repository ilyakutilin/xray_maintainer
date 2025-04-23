package utils

import (
	"strings"
)

type Errors []error

func (e Errors) Error() string {
	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func (e *Errors) Append(err error) {
	if err != nil {
		*e = append(*e, err)
	}
}

func (e Errors) IsEmpty() bool {
	return len(e) == 0
}
