package messages

import "fmt"

// StreamSender implements Sender for debugging (to stdout)
type StreamSender struct{}

// Send sends a message using the StreamSender. It prints the message details,
// including the subject, body, and any associated errors, to the standard output.
// If there are no errors, it indicates so. If there is one error, it prints the error.
// If there are multiple errors, it lists them all.
//
// Parameters:
//   - msg: The Message to be sent, containing a subject, body, and a list of errors.
//
// Returns:
//   - error: Always returns nil as this implementation does not perform actual sending.
func (s *StreamSender) Send(msg Message) error {
	if msg.String() != "" {
		fmt.Println("The following message would be sent:")
		fmt.Printf("%s", msg)
	}

	return nil
}

func (s *StreamSender) Validate() error {
	// No validation needed for StreamSender
	return nil
}
