package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"time"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
)

var ErrNoValidFromAddress = errors.New("SendMessage: no valid sender address")

func (s *EmailService) SendMessage(ctx context.Context, email Email) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("SendMessage: context canceled: %w", ctx.Err())
	default:
	}

	msg := gomail.NewMessage()

	_, err := mail.ParseAddress(s.config.SenderEmail)
	if err != nil {
		return ErrNoValidFromAddress
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
		InsecureSkipVerify: false,
	}

	s.logger.Info(fmt.Sprintf("SendMessage: sending email to %s", email.To))

	if err := s.sendWithRetry(ctx, dialer, msg); err != nil {
		s.logger.Error(fmt.Sprintf("SendMessage: cannot send message to %s", email.To), zap.Error(err))
		return fmt.Errorf("sendMessage: cannot send message to %s, %w", email.To, err)
	}

	s.logger.Info(fmt.Sprintf("SendMessage: successfully sent message to %s", email.To))
	return nil
}

func (s *EmailService) sendWithRetry(ctx context.Context, dialer *gomail.Dialer, msg *gomail.Message) error {
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
