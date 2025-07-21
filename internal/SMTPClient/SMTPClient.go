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

// New creates and returns a new SMTPClient instance.
// If MaxRetries and BasicRetryPause are not set in the configuration, the default values are applied.
func New(config *Config, metrics monitoring.Monitoring, logger *zap.Logger) *SMTPClient {
	if config.MaxRetries == 0 {
		config.MaxRetries = DefaultMaxRetries
	}

	if config.BasicRetryPause == 0 {
		config.BasicRetryPause = DefaultBasicRetryPause
	}

	return &SMTPClient{
		config:  config,
		metrics: metrics,
		logger:  logger,
	}
}

// SendEmail sends the provided email using a Simple Mail Transfer Protocol (SMTP).
// If sending false, it reties using exponential backoff.
func (s *SMTPClient) SendEmail(ctx context.Context, email EmailMessage) error {
	if ctx.Err() != nil {
		s.metrics.IncCanceled("SendEmail")
		s.logger.Error(ErrContextCanceledBeforeSending.Error(), zap.Error(ctx.Err()))

		return ErrContextCanceledBeforeSending
	}

	start := time.Now()

	msg := gomail.NewMessage()

	_, err := mail.ParseAddress(s.config.SenderEmail)
	if err != nil {
		s.metrics.IncError("SendEmail")
		s.logger.Error("SendEmail: no valid sender address", zap.Error(err))

		return ErrNoValidSenderAddress
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
		s.metrics.IncError("SendEmail")
		s.logger.Error(fmt.Sprintf("SendEmail: cannot send message to %s", email.To), zap.Error(err))

		return fmt.Errorf("SendEmail: cannot send message to %s, %w", email.To, err)
	}

	s.logger.Info(fmt.Sprintf("SendEmail: successfully sent message to %s", email.To))

	s.metrics.Observe("SendEmail", start)
	s.metrics.IncSuccess("SendEmail")

	return nil
}

// sendWithRetry attempts to send the email using the provided dialer with exponential backoff retries.
func (s *SMTPClient) sendWithRetry(ctx context.Context, dialer *gomail.Dialer, msg *gomail.Message) error {
	var lastErr error

	for i := 0; i < s.config.MaxRetries+1; i++ {
		if ctx.Err() != nil {
			s.metrics.IncCanceled("SendEmail")
			s.logger.Error(ErrContextCanceledBeforeRetry.Error(), zap.Error(ctx.Err()))

			return ErrContextCanceledBeforeRetry
		}

		if i > 0 {
			pause := s.CreatePause(i)
			s.logger.Info(
				"sendWithRetry: retrying send message",
				zap.Int("attempt", i),
				zap.Duration("pause", pause),
				zap.Error(lastErr),
			)

			select {
			case <-time.After(pause):
			case <-ctx.Done():
				s.metrics.IncCanceled("SendEmail")
				s.logger.Error(ErrContextCanceledAfterPause.Error(), zap.Error(ctx.Err()))

				return ErrContextCanceledAfterPause
			}
		}

		err := dialer.DialAndSend(msg)
		if err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	s.logger.Error("sendWithRetry: all attempts to send message failed, last error:", zap.Error(lastErr))
	return fmt.Errorf("sendWithRetry: all attempts to send message failed, last error: %w", lastErr)
}

// CreatePause calculates the delay before the next retry attempt using the formula:
// basePause * 2^(retryAttempt - 1), implementing exponential backoff.
func (s *SMTPClient) CreatePause(i int) time.Duration {
	pauseFloat := s.config.BasicRetryPause.Seconds() * math.Pow(2, float64(i-1))
	pause := time.Duration(pauseFloat) * time.Second

	return pause
}
