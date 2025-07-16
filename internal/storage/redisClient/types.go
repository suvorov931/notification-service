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

const (
	DefaultRedisTimeout = 3 * time.Second
)

type Config struct {
	Addrs    []string      `yaml:"REDIS_CLUSTER_ADDRS" env:"REDIS_CLUSTER_ADDRS"`
	Timeout  time.Duration `yaml:"REDIS_CLUSTER_TIMEOUT" env:"REDIS_CLUSTER_TIMEOUT"`
	Password string        `yaml:"REDIS_CLUSTER_PASSWORD" env:"REDIS_CLUSTER_PASSWORD"`
	ReadOnly bool          `yaml:"REDIS_CLUSTER_READ_ONLY" env:"REDIS_CLUSTER_READ_ONLY"`
}

type RedisCluster struct {
	cluster *redis.ClusterClient
	metrics monitoring.Monitoring
	logger  *zap.Logger
	timeout time.Duration
}

type RedisClient interface {
	AddDelayedEmail(context.Context, *SMTPClient.EmailMessage) error
	CheckRedis(context.Context) ([]string, error)
	Close() error
}

type MockRedisClient struct {
	mock.Mock
}

func (mrc *MockRedisClient) AddDelayedEmail(ctx context.Context, email *SMTPClient.EmailMessage) error {
	args := mrc.Called(ctx, email)
	return args.Error(0)
}

func (mrc *MockRedisClient) CheckRedis(ctx context.Context) ([]string, error) {
	args := mrc.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (mrc *MockRedisClient) Close() error {
	args := mrc.Called()
	return args.Error(0)
}
