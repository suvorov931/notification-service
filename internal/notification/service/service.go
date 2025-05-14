package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

func (s *MailService) SendMessage(ctx context.Context, mail Mail) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("SendMessage: context canceled: %w", ctx.Err())
	default:
	}

	msg := gomail.NewMessage()

	msg.SetHeader("From", s.config.SenderEmail)
	msg.SetHeader("To", mail.To)
	msg.SetHeader("Subject", mail.Subject)
	msg.SetBody("text/plain", mail.Message)

	dialer := gomail.NewDialer(
		s.config.SMTPHost,
		s.config.SMTPPort,
		s.config.SenderEmail,
		s.config.SenderPassword,
	)

	dialer.TLSConfig = &tls.Config{
		ServerName:         s.config.SMTPHost,
		InsecureSkipVerify: false,
	}

	s.logger.Info(fmt.Sprintf("SendMessage: sending email to %s", mail.To))

	if err := s.sendWithRetry(ctx, dialer, msg); err != nil {
		s.logger.Error(fmt.Sprintf("SendMessage: cannot send message to %s", mail.To), zap.Error(err))
		return fmt.Errorf("sendMessage: cannot send message to %s, %w", mail.To, err)
	}

	s.logger.Info(fmt.Sprintf("SendMessage: successfully sent message to %s", mail.To))
	return nil
}

func (s *MailService) sendWithRetry(ctx context.Context, dialer *gomail.Dialer, msg *gomail.Message) error {
	var lastErr error

	for i := 0; i < maxRetries+1; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("SendMessage: context canceled: %w", ctx.Err())
		default:
		}

		if i > 0 {
			pause := time.Duration(basicRetryPause*math.Pow(2, float64(i-1))) * time.Second
			s.logger.Info(
				"SendMessage: retrying send message",
				zap.Int("attempt", i),
				zap.Duration("pause", pause),
				zap.Error(lastErr),
			)
			select {
			case <-time.After(pause):
			case <-ctx.Done():
				return fmt.Errorf("SendMessage: context canceled: %w", ctx.Err())
			}
		}

		if err := dialer.DialAndSend(msg); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	s.logger.Error("SendMessage: all attempts to send message failed", zap.Error(lastErr))
	return fmt.Errorf("sendMessage: all attempts to send message failed, %w", lastErr)
}
