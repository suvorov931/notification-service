package sendMessage

import (
	"fmt"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
	"notification/internal/config"
)

func SendMessage(cfg *config.Config, logger *zap.Logger, to string, subject string, message string) error {
	msg := gomail.NewMessage()

	msg.SetHeader("From", cfg.SenderEmail)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", message)

	dialer := gomail.NewDialer("smtp.mail.ru", 587, cfg.SenderEmail, cfg.SenderPassword)

	if err := dialer.DialAndSend(msg); err != nil {
		logger.Error(CannotSendMessage(to, err))
		return fmt.Errorf(CannotSendMessage(to, err))
	}

	logger.Info("Successfully sent message")
	return nil
}

func CannotSendMessage(to string, err error) string {
	return fmt.Sprintf("cannot send message to %s : %v", to, err)
}
