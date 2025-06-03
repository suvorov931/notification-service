package service

import (
	"context"

	"go.uber.org/zap"
)

type MailSender struct {
	SenderEmail     string `yaml:"SENDER_EMAIL" env:"SENDER_EMAIL"`
	SenderPassword  string `yaml:"SENDER_PASSWORD" env:"SENDER_PASSWORD"`
	SMTPHost        string `yaml:"SMTP_HOST" env:"SMTP_HOST"`
	SMTPPort        int    `yaml:"SMTP_PORT" env:"SMTP_PORT"`
	SkipVerify      bool   `yaml:"SKIP_VERIFY" env:"SKIP_VERIFY"`
	MaxRetries      int    `yaml:"MAX_RETRIES" env:"MAX_RETRIES" env-default:"3"`
	BasicRetryPause int    `yaml:"BASIC_RETRY_PAUSE" env:"BASIC_RETRY_PAUSE" env-default:"5"`
}

type Email struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type TempEmailWithTime struct {
	Time    string `json:"time"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type EmailWithTime struct {
	Time  string
	Email Email
}

type EmailService struct {
	config *MailSender
	logger *zap.Logger
}

type EmailSender interface {
	SendMessage(ctx context.Context, email Email) error
}

func New(config *MailSender, logger *zap.Logger) *EmailService {
	return &EmailService{
		config: config,
		logger: logger,
	}
}
