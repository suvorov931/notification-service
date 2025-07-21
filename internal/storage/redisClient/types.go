package redisClient

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/monitoring"
)

// DefaultRedisTimeout defines the default timeout for Redis operations.
const DefaultRedisTimeout = 3 * time.Second

// Config defines the configuration parameters for the RedisCluster,
// including cluster addresses, credentials, and timeout settings.
type Config struct {
	Addrs           []string      `env:"REDIS_CLUSTER_ADDRS"`
	Timeout         time.Duration `env:"REDIS_CLUSTER_TIMEOUT"`
	ShutdownTimeout time.Duration `env:"REDIS_CLUSTER_SHUTDOWN_TIMEOUT"`
	Password        string        `env:"REDIS_CLUSTER_PASSWORD"`
	ReadOnly        bool          `env:"REDIS_CLUSTER_READ_ONLY"`
}

// RedisCluster implements the RedisClient interface.
// It provides methods for saving and retrieving emails using a Redis database.
type RedisCluster struct {
	cluster         *redis.ClusterClient
	metrics         monitoring.Monitoring
	logger          *zap.Logger
	timeout         time.Duration
	shutdownTimeout time.Duration
}

// RedisClient defines an interface for saving and retrieving emails in a Redis database.
type RedisClient interface {
	AddDelayedEmail(context.Context, *SMTPClient.EmailMessage) error
	CheckRedis(context.Context) ([]string, error)
	Close() error
}

// MockRedisClient is a mock implementation of the RedisClient interface,
// used for testing components that interact with the database layer.
type MockRedisClient struct {
	mock.Mock
}

// AddDelayedEmail is a mock implementation.
func (mrc *MockRedisClient) AddDelayedEmail(ctx context.Context, email *SMTPClient.EmailMessage) error {
	args := mrc.Called(ctx, email)
	return args.Error(0)
}

// CheckRedis is a mock implementation.
func (mrc *MockRedisClient) CheckRedis(ctx context.Context) ([]string, error) {
	args := mrc.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

// Close is a mock implementation.
func (mrc *MockRedisClient) Close() error {
	args := mrc.Called()
	return args.Error(0)
}
