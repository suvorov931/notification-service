package SMTPClient

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/mail"
	"time"

	"go.uber.org/zap"
	"gopkg.in/gomail.v2"

	"notification/internal/monitoring"
)

func New(config *Config, metrics monitoring.Monitoring, logger *zap.Logger) *SMTPClient {
	return &SMTPClient{
		config:  config,
		metrics: metrics,
		logger:  logger,
	}
}

func (s *SMTPClient) SendEmail(ctx context.Context, email EmailMessage) error {
	if ctx.Err() != nil {
		s.logger.Error("SendEmail: context canceled", zap.Error(ctx.Err()))
		return fmt.Errorf("SendEmail: context canceled")
	}

	start := time.Now()

	msg := gomail.NewMessage()

	_, err := mail.ParseAddress(s.config.SenderEmail)
	if err != nil {
		s.metrics.Inc("SendEmail", monitoring.StatusError)
		s.logger.Error("SendEmail: no valid sender address", zap.Error(err))
		return fmt.Errorf("SendEmail: no valid sender address")
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

	s.logger.Info(fmt.Sprintf("SendEmail: sending email to %s", email.To))

	if err = s.sendWithRetry(ctx, dialer, msg); err != nil {
		s.metrics.Inc("SendEmail", monitoring.StatusError)
		s.logger.Error(fmt.Sprintf("SendEmail: cannot send message to %s", email.To), zap.Error(err))
		return fmt.Errorf("SendEmail: cannot send message to %s, %w", email.To, err)
	}

	s.logger.Info(fmt.Sprintf("SendEmail: successfully sent message to %s", email.To))

	duration := time.Since(start).Seconds()
	s.metrics.Observe("SendEmail", duration)
	s.metrics.Inc("SendEmail", monitoring.StatusSuccess)

	return nil
}

func (s *SMTPClient) sendWithRetry(ctx context.Context, dialer *gomail.Dialer, msg *gomail.Message) error {
	var lastErr error

	for i := 0; i < s.config.MaxRetries+1; i++ {
		if ctx.Err() != nil {
			s.metrics.Inc("SendEmail", monitoring.StatusCanceled)
			s.logger.Error("sendWithRetry: context canceled", zap.Error(ctx.Err()))
			return fmt.Errorf("sendWithRetry: context canceled")
		}

		if i > 0 {
			pause := time.Duration(float64(s.config.BasicRetryPause)*math.Pow(2, float64(i-1))) * time.Second
			s.logger.Info(
				"sendWithRetry: retrying send message",
				zap.Int("attempt", i),
				zap.Duration("pause", pause),
				zap.Error(lastErr),
			)
			select {
			case <-time.After(pause):
			case <-ctx.Done():
				s.metrics.Inc("SendEmail", monitoring.StatusCanceled)
				s.logger.Error("sendWithRetry: context canceled", zap.Error(ctx.Err()))
				return fmt.Errorf("sendWithRetry: context canceled")
			}
		}

		if err := dialer.DialAndSend(msg); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	s.logger.Error("sendWithRetry: all attempts to send message failed, last error:", zap.Error(lastErr))
	return fmt.Errorf("sendWithRetry: all attempts to send message failed, last error: %w", lastErr)
}
