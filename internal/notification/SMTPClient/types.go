package SMTPClient

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"notification/internal/monitoring"
)

type Config struct {
	SenderEmail     string `yaml:"SENDER_EMAIL" env:"SENDER_EMAIL"`
	SenderPassword  string `yaml:"SENDER_PASSWORD" env:"SENDER_PASSWORD"`
	SMTPHost        string `yaml:"SMTP_HOST" env:"SMTP_HOST"`
	SMTPPort        int    `yaml:"SMTP_PORT" env:"SMTP_PORT"`
	SkipVerify      bool   `yaml:"SKIP_VERIFY" env:"SKIP_VERIFY"`
	MaxRetries      int    `yaml:"MAX_RETRIES" env:"MAX_RETRIES"`
	BasicRetryPause int    `yaml:"BASIC_RETRY_PAUSE" env:"BASIC_RETRY_PAUSE"`
}

type EmailMessage struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type TempEmailMessageWithTime struct {
	Time    string `json:"time"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type EmailMessageWithTime struct {
	Time  string
	Email EmailMessage
}

type EmailSender interface {
	SendEmail(ctx context.Context, email EmailMessage) error
}

type SMTPClient struct {
	config  *Config
	metrics monitoring.Monitoring
	logger  *zap.Logger
}

func New(config *Config, metrics monitoring.Monitoring, logger *zap.Logger) *SMTPClient {
	return &SMTPClient{
		config:  config,
		metrics: metrics,
		logger:  logger,
	}
}

type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendEmail(ctx context.Context, email EmailMessage) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}
