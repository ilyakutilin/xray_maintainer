package messages

// EmailSender implements Sender for email
type EmailSender struct {
	// TODO: email-specific config (SMTP server, credentials, etc.)
}

func (e *EmailSender) Send(msg Message) error {
	// TODO: implementation for sending email
	return nil
}

func (e *EmailSender) Validate() error {
	// TODO: validate email-specific config
	return nil
}
