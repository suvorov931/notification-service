package postgresClient

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/monitoring"
)

const (
	queryForAddInstantSending = `INSERT INTO schema_emails.instant_sending ("to", subject,message)
	VALUES ($1, $2, $3) RETURNING id`
	queryForAddDelayedSending = `INSERT INTO schema_emails.delayed_sending (time, "to", subject,message)
	VALUES ($1, $2, $3, $4) RETURNING id`
)

type Config struct {
	Host     string `env:"POSTGRES_HOST"`
	Port     string `env:"POSTGRES_PORT"`
	User     string `env:"POSTGRES_USER"`
	Password string `env:"POSTGRES_PASSWORD"`
	Database string `env:"POSTGRES_DATABASE"`
	MaxConns int    `env:"POSTGRES_MAX_CONNECTIONS"`
	MinConns int    `env:"POSTGRES_MIN_CONNECTIONS"`
}

type PostgresService struct {
	pool    *pgxpool.Pool
	metrics monitoring.Monitoring
	logger  *zap.Logger
}

type PostgresClient interface {
	SavingInstantSending(context.Context, *SMTPClient.EmailMessage) (int, error)
	SavingDelayedSending(context.Context, *SMTPClient.EmailMessageWithTime) (int, error)
}

type MockPostgresService struct {
	mock.Mock
}

func (mps *MockPostgresService) SavingInstantSending(ctx context.Context, email *SMTPClient.EmailMessage) (int, error) {
	args := mps.Called(ctx, email)
	return args.Get(0).(int), args.Error(1)
}
func (mps *MockPostgresService) SavingDelayedSending(ctx context.Context, email *SMTPClient.EmailMessageWithTime) (int, error) {
	args := mps.Called(ctx, email)
	return args.Get(0).(int), args.Error(1)
}
