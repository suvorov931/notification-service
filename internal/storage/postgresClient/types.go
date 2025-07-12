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
	SaveEmail(context.Context, *SMTPClient.EmailMessage) (int, error)
	FetchById(context.Context, int) ([]*SMTPClient.EmailMessage, error)
	FetchByEmail(context.Context, string) ([]*SMTPClient.EmailMessage, error)
	FetchByAll(context.Context) ([]*SMTPClient.EmailMessage, error)
	Close()
}

type MockPostgresService struct {
	mock.Mock
}

func (mps *MockPostgresService) SaveEmail(ctx context.Context, email *SMTPClient.EmailMessage) (int, error) {
	args := mps.Called(ctx, email)
	return args.Get(0).(int), args.Error(1)
}

func (mps *MockPostgresService) FetchById(ctx context.Context, id int) ([]*SMTPClient.EmailMessage, error) {
	args := mps.Called(ctx, id)
	return args.Get(0).([]*SMTPClient.EmailMessage), args.Error(1)
}

func (mps *MockPostgresService) FetchByEmail(ctx context.Context, email string) ([]*SMTPClient.EmailMessage, error) {
	args := mps.Called(ctx, email)
	return args.Get(0).([]*SMTPClient.EmailMessage), args.Error(1)
}

func (mps *MockPostgresService) FetchByAll(ctx context.Context) ([]*SMTPClient.EmailMessage, error) {
	args := mps.Called(ctx)
	return args.Get(0).([]*SMTPClient.EmailMessage), args.Error(1)
}

func (mps *MockPostgresService) Close() {}
