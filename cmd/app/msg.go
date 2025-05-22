package main

import (
	"github.com/ilyakutilin/xray_maintainer/messages"
)

func (app *Application) getSender(msgCfg Messages) messages.Sender {
	var sender messages.Sender

	if !app.debug {
		rawSenders := []messages.Sender{
			&msgCfg.EmailSender,
			&msgCfg.TelegramSender,
		}

		var validSenders []messages.Sender

		for _, sdr := range rawSenders {
			if err := sdr.Validate(); err != nil {
				app.logger.Warning.Printf("The sender failed validation and "+
					"will not be included in the senders list: %v", err)
			}
			validSenders = append(validSenders, sdr)
		}

		if validSenders != nil {
			sender = &messages.CompositeSender{
				Senders: validSenders,
			}
		}
	}

	if sender == nil {
		sender = &msgCfg.StreamSender
	}

	return sender
}

func (app *Application) sendMsg(msgCfg Messages, subject, body string) {
	sender := app.getSender(msgCfg)

	message := messages.Message{
		Subject:  subject,
		Body:     body,
		Notes:    app.notes,
		Warnings: app.warnings,
	}

	if err := sender.Send(message); err != nil {
		app.logger.Warning.Printf("Failed to send this message:\n"+
			"%s\nReason: %v", message, err)
	}
}
