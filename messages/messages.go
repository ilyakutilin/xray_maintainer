package messages

import (
	"fmt"
	"strings"

	"github.com/ilyakutilin/xray_maintainer/utils"
)

type Message struct {
	Subject  string
	Body     string
	Notes    []string
	Warnings []string
}

func (m Message) getFullBody(html bool) string {
	var bs []string

	if m.Body != "" {
		bs = append(bs, m.Body)
	}

	var opTag string
	var clTag string

	if html {
		opTag = "<b>"
		clTag = "</b>"
	}

	notesCount := len(m.Notes)
	switch notesCount {
	case 0:
	case 1:
		bs = append(bs, fmt.Sprintf("%sNote%s: %s", opTag, clTag, m.Notes[0]))
	default:
		var notes []string
		notes = append(notes, fmt.Sprintf("%sNotes%s:", opTag, clTag))
		for i, note := range m.Notes {
			notes = append(notes, fmt.Sprintf("%s%d)%s %s", opTag, i+1, clTag, note))
		}
		bs = append(bs, strings.Join(notes, "\n"))
	}

	warningsCount := len(m.Warnings)
	switch warningsCount {
	case 0:
	case 1:
		bs = append(bs, fmt.Sprintf("%sWarning%s: %s", opTag, clTag, m.Warnings[0]))
	default:
		var warnings []string
		warnings = append(warnings, fmt.Sprintf("%sWarnings%s:", opTag, clTag))
		for i, warning := range m.Warnings {
			warnings = append(warnings, fmt.Sprintf("%s%d)%s %s", opTag, i+1, clTag, warning))
		}
		bs = append(bs, strings.Join(warnings, "\n"))
	}

	res := strings.Join(bs, "\n\n")

	if res != "" && res[len(res)-1] != '\n' {
		res += "\n"
	}
	return res
}

func (m Message) GetFullBodyText() string {
	return m.getFullBody(false)
}

func (m Message) GetFullBodyHTML() string {
	return m.getFullBody(true)
}

func (m Message) String() string {
	if m.Subject == "" && m.Body == "" {
		return ""
	}

	if m.Subject == "" {
		m.Subject = "[empty subject]"
	}

	if m.Body == "" {
		m.Body = "[empty body]"
	}

	return fmt.Sprintf("Subject: %s\n", m.Subject) +
		fmt.Sprintf("Body: %s", m.GetFullBodyText())
}

type Sender interface {
	Send(msg Message) error
	Validate() error
}

type CompositeSender struct {
	Senders []Sender
}

// Send attempts to send the provided message using all the senders in the
// CompositeSender. It first validates each sender before attempting to send the
// message. If a sender fails validation, it skips sending for that sender and records
// the validation error. If sending// the message fails for a sender, the error is also
// recorded.
//
// Returns a utils.Errors object containing all validation and sending errors if any
// occurred, or nil if the message was successfully sent by all valid senders.
func (c *CompositeSender) Send(msg Message) error {
	var errs utils.Errors
	for _, s := range c.Senders {
		if err := s.Validate(); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := s.Send(msg); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// Validate checks the validity of each sender in the CompositeSender.
// If any sender fails validation, it records the error and continues to check
// the remaining senders. Returns a utils.Errors object containing all validation
// errors if any occurred, or nil if all senders are valid.
func (c *CompositeSender) Validate() error {
	var errs utils.Errors
	for _, s := range c.Senders {
		if err := s.Validate(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}
