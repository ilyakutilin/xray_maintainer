package messages

import "github.com/ilyakutilin/xray_maintainer/utils"

type Message struct {
	Subject  string
	Body     string
	Notes    []string
	Warnings []string
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
