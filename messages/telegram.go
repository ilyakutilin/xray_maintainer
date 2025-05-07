package messages

// TelegramSender implements Sender for Telegram
type TelegramSender struct {
	// TODO: telegram-specific config (bot token, chat ID, etc.)
}

func (t *TelegramSender) Send(msg Message) error {
	// TODO: implementation for sending Telegram message
	return nil
}

func (t *TelegramSender) Validate() error {
	// TODO: validate Telegram-specific config
	return nil
}
