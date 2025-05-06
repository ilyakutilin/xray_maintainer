package messages

// TelegramSender implements Sender for Telegram
type TelegramSender struct {
	// telegram-specific config (bot token, chat ID, etc.)
}

func (t *TelegramSender) Send(msg Message) error {
	// implementation for sending Telegram message
	return nil
}
