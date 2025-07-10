package postgresClient

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/monitoring"
)

const (
	DefaultPostgresTimeout = 3 * time.Second
)

type Config struct {
	Host     string        `env:"POSTGRES_HOST"`
	Port     string        `env:"POSTGRES_PORT"`
	User     string        `env:"POSTGRES_USER"`
	Password string        `env:"POSTGRES_PASSWORD"`
	Database string        `env:"POSTGRES_DATABASE"`
	Timeout  time.Duration `env:"POSTGRES_TIMEOUT"`
	MaxConns int           `env:"POSTGRES_MAX_CONNECTIONS"`
	MinConns int           `env:"POSTGRES_MIN_CONNECTIONS"`
}

type PostgresService struct {
	pool    *pgxpool.Pool
	metrics monitoring.Monitoring
	logger  *zap.Logger
	timeout time.Duration
}

type PostgresClient interface {
	SaveEmail(context.Context, any) (int, error)
	FetchById(context.Context, string) (any, error)
	FetchByMail(context.Context, string) ([]any, error)
	FetchByAll(context.Context, string) ([]any, error)
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
