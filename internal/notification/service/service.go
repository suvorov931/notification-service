package mail

import (
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

const (
	maxRetries      = 3
	basicRetryPause = 5
)

func (s *MailService) SendMessage(mail Mail) error {
	msg := gomail.NewMessage()

	msg.SetHeader("From", s.config.CredentialsSender.SenderEmail)
	msg.SetHeader("To", mail.To)
	msg.SetHeader("Subject", mail.Subject)
	msg.SetBody("text/plain", mail.Message)

	dialer := gomail.NewDialer(
		s.config.CredentialsSender.SMTPHost,
		s.config.CredentialsSender.SMTPPORT,
		s.config.CredentialsSender.SenderEmail,
		s.config.CredentialsSender.SenderPassword,
	)

	s.logger.Info(fmt.Sprintf("Send message: sending email to %s", mail.To))

	if err := s.sendWithRetry(dialer, msg); err != nil {
		s.logger.Error(fmt.Sprintf("send message: cannot send message to %s", mail.To), zap.Error(err))
		return fmt.Errorf("send message: cannot send message to %s, %w", mail.To, err)
	}

	s.logger.Info(fmt.Sprintf("Send message: successfully sent message to %s", mail.To))
	return nil
}

func (s *MailService) sendWithRetry(dialer *gomail.Dialer, msg *gomail.Message) error {
	var lastErr error

	for i := 0; i < maxRetries+1; i++ {
		if i > 0 {
			pause := time.Duration(basicRetryPause*math.Pow(2, float64(i-1))) * time.Second
			s.logger.Info(
				"Send message: retrying send message",
				zap.Int("attempt", i),
				zap.Duration("pause", pause),
				zap.Error(lastErr),
			)
			time.Sleep(pause)
		}

		if err := dialer.DialAndSend(msg); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	s.logger.Error("Send message: all attempts to send message failed", zap.Error(lastErr))
	return fmt.Errorf("all attempts to send message failed, %w", lastErr)
}
