package redisClient

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"notification/internal/monitoring"
)

const (
	DefaultRedisTimeout = 3 * time.Second
	emailTimeLayout     = "2006-01-02 15:04:05"
)

type Config struct {
	Addrs    []string      `yaml:"REDIS_CLUSTER_ADDRS"`
	Timeout  time.Duration `yaml:"REDIS_CLUSTER_TIMEOUT"`
	Password string        `yaml:"REDIS_CLUSTER_PASSWORD"`
	ReadOnly bool          `yaml:"REDIS_CLUSTER_READ_ONLY"`
}

type RedisCluster struct {
	cluster *redis.ClusterClient
	metrics monitoring.Monitoring
	logger  *zap.Logger
	timeout time.Duration
}

type MockRedisClient struct {
	mock.Mock
}

func (mrc *MockRedisClient) CheckRedis(ctx context.Context) ([]string, error) {
	args := mrc.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}
