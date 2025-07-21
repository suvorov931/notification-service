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
	// DefaultMaxRetries is the default value for MaxRetries.
	DefaultMaxRetries = 3

	// DefaultBasicRetryPause is the default value for BasicRetryPause.
	DefaultBasicRetryPause = 5 * time.Second
)

var (
	// ErrNoValidSenderAddress indicates that the sender address is invalid.
	ErrNoValidSenderAddress = fmt.Errorf("SendEmail: no valid sender address")

	// ErrContextCanceledBeforeSending indicates that the context was canceled before the email was sent.
	ErrContextCanceledBeforeSending = fmt.Errorf("SendEmail: context canceled before sending")

	// ErrContextCanceledBeforeRetry indicates that the context was canceled before retrying the email send.
	ErrContextCanceledBeforeRetry = fmt.Errorf("sendWithRetry: context canceled before sending")

	// ErrContextCanceledAfterPause indicates that the context was canceled after the retry pause but before sending.
	ErrContextCanceledAfterPause = fmt.Errorf("sendWithRetry: context canceled after pause")
)

// Config defines the configuration parameters for the SMTPClient,
// including sender credentials, timeout and retry configuration.
type Config struct {
	SenderEmail     string        `env:"SENDER_EMAIL"`
	SenderPassword  string        `env:"SENDER_PASSWORD"`
	SMTPHost        string        `env:"SMTP_HOST"`
	SMTPPort        int           `env:"SMTP_PORT"`
	SkipVerify      bool          `env:"SKIP_VERIFY"`
	MaxRetries      int           `env:"MAX_RETRIES"`
	BasicRetryPause time.Duration `env:"BASIC_RETRY_PAUSE"`
}

// TempEmailMessage is used as an intermediate structure for decode from/to JSON.
type TempEmailMessage struct {
	Type    string `json:"type"`
	Time    string `json:"time"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// EmailMessage contains the email details, including an optional Time field for delayed delivery.
type EmailMessage struct {
	Type    string     `json:"type"`
	Time    *time.Time `json:"time,omitempty"`
	To      string     `json:"to"`
	Subject string     `json:"subject"`
	Message string     `json:"message"`
}

// SMTPClient implements the EmailSender interface and sends email messages using SMTP.
type SMTPClient struct {
	config  *Config
	metrics monitoring.Monitoring
	logger  *zap.Logger
}

// EmailSender defines an interface for sending email messages to recipient.
type EmailSender interface {
	SendEmail(context.Context, EmailMessage) error
	CreatePause(int) time.Duration
}

// MockEmailSender is a mock implementation of the EmailSender interface,
// used for testing components that depend on email sending behavior.
type MockEmailSender struct {
	mock.Mock
}

// SendEmail is a mock implementation.
func (m *MockEmailSender) SendEmail(ctx context.Context, email EmailMessage) error {
	args := m.Called(ctx, email)
	return args.Error(0)
}

// CreatePause is a mock implementation.
func (m *MockEmailSender) CreatePause(i int) time.Duration {
	return time.Second
}
