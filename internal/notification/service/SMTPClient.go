package service

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/mail"
	"time"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

func (s *SMTPClient) SendEmail(ctx context.Context, email EmailMessage) error {
	select {
	case <-ctx.Done():
		s.logger.Error("SendMessage: context canceled", zap.Error(ctx.Err()))
		return fmt.Errorf("SendMessage: context canceled")
	default:
	}

	msg := gomail.NewMessage()

	_, err := mail.ParseAddress(s.config.SenderEmail)
	if err != nil {
		s.logger.Error("SendMessage: no valid sender address", zap.Error(err))

		return fmt.Errorf("SendMessage: no valid sender address")
	}

	msg.SetHeader("From", s.config.SenderEmail)
	msg.SetHeader("To", email.To)
	msg.SetHeader("Subject", email.Subject)
	msg.SetBody("text/plain", email.Message)

	dialer := gomail.NewDialer(
		s.config.SMTPHost,
		s.config.SMTPPort,
		s.config.SenderEmail,
		s.config.SenderPassword,
	)

	dialer.TLSConfig = &tls.Config{
		ServerName:         s.config.SMTPHost,
		InsecureSkipVerify: s.config.SkipVerify,
	}

	s.logger.Info(fmt.Sprintf("SendMessage: sending email to %s", email.To))

	if err = s.sendWithRetry(ctx, dialer, msg); err != nil {
		s.logger.Error(fmt.Sprintf("SendMessage: cannot send message to %s", email.To), zap.Error(err))
		return fmt.Errorf("SendMessage: cannot send message to %s, %w", email.To, err)
	}

	s.logger.Info(fmt.Sprintf("SendMessage: successfully sent message to %s", email.To))
	return nil
}

func (s *SMTPClient) sendWithRetry(ctx context.Context, dialer *gomail.Dialer, msg *gomail.Message) error {
	var lastErr error

	for i := 0; i < s.config.MaxRetries+1; i++ {
		select {
		case <-ctx.Done():
			s.logger.Error("SendMessage: context canceled", zap.Error(ctx.Err()))
			return fmt.Errorf("SendMessage: context canceled")
		default:
		}

		if i > 0 {
			pause := time.Duration(float64(s.config.BasicRetryPause)*math.Pow(2, float64(i-1))) * time.Second
			s.logger.Info(
				"SendMessage: retrying send message",
				zap.Int("attempt", i),
				zap.Duration("pause", pause),
				zap.Error(lastErr),
			)
			select {
			case <-time.After(pause):
			case <-ctx.Done():
				s.logger.Error("SendMessage: context canceled", zap.Error(ctx.Err()))
				return fmt.Errorf("SendMessage: context canceled")
			}
		}

		if err := dialer.DialAndSend(msg); err != nil {
			lastErr = err
			continue
		}

		return nil
	}
	s.logger.Error("SendMessage: all attempts to send message failed, last error:", zap.Error(lastErr))
	return fmt.Errorf("SendMessage: all attempts to send message failed, last error: %w", lastErr)
}
