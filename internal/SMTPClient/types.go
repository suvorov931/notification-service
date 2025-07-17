package SMTPClient

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"notification/internal/monitoring"
)

const (
	DefaultMaxRetries      = 3
	DefaultBasicRetryPause = 5 * time.Second
)

var (
	ErrNoValidSenderAddress         = fmt.Errorf("SendEmail: no valid sender address")
	ErrContextCanceledBeforeSending = fmt.Errorf("SendEmail: context canceled before sending")
	ErrContextCanceledBeforeRetry   = fmt.Errorf("sendWithRetry: context canceled before sending")
	ErrContextCanceledAfterPause    = fmt.Errorf("sendWithRetry: context canceled after pause")
)

type Config struct {
	SenderEmail     string        `env:"SENDER_EMAIL"`
	SenderPassword  string        `env:"SENDER_PASSWORD"`
	SMTPHost        string        `env:"SMTP_HOST"`
	SMTPPort        int           `env:"SMTP_PORT"`
	SkipVerify      bool          `env:"SKIP_VERIFY"`
	MaxRetries      int           `env:"MAX_RETRIES"`
	BasicRetryPause time.Duration `env:"BASIC_RETRY_PAUSE"`
}

type TempEmailMessage struct {
	Type    string `json:"type"`
	Time    string `json:"time"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type EmailMessage struct {
	Type    string     `json:"type"`
	Time    *time.Time `json:"time,omitempty"`
	To      string     `json:"to"`
	Subject string     `json:"subject"`
	Message string     `json:"message"`
}

type EmailSender interface {
	SendEmail(context.Context, EmailMessage) error
	CreatePause(int) time.Duration
}

type SMTPClient struct {
	config  *Config
	metrics monitoring.Monitoring
	logger  *zap.Logger
}

type MockEmailSender struct {
	mock.Mock
}

func (m *MockEmailSender) SendEmail(ctx context.Context, email EmailMessage) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

func (m *MockEmailSender) CreatePause(i int) time.Duration {
	return time.Second
}
