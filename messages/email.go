package messages

// EmailSender implements Sender for email
type EmailSender struct {
	// email-specific config (SMTP server, credentials, etc.)
}

func (e *EmailSender) Send(msg Message) error {
	// implementation for sending email
	return nil
}
