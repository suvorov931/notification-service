package rds

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"notification/internal/monitoring"
	"notification/internal/notification/SMTPClient"
	"notification/internal/notification/api"
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
	Cluster *redis.ClusterClient
	Logger  *zap.Logger
	Timeout time.Duration
}

// TODO: rename package to RedisCluster/RedisClient?

func New(ctx context.Context, cfg *Config, logger *zap.Logger) (*RedisCluster, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultRedisTimeout
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	cluster := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    cfg.Addrs,
		Password: cfg.Password,
		ReadOnly: cfg.ReadOnly,
	})

	if err := cluster.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis cluster: %w", err)
	}

	return &RedisCluster{
		Cluster: cluster,
		Logger:  logger,
		Timeout: cfg.Timeout,
	}, nil
}

func (rc *RedisCluster) AddDelayedEmail(ctx context.Context, email *SMTPClient.EmailMessageWithTime) error {
	ctx, cancel := context.WithTimeout(ctx, rc.Timeout)
	defer cancel()

	start := time.Now()

	emailJSON, score, err := rc.parseAndConvertTime(email)
	if err != nil {
		monitoring.RedisErrorCounter.Inc()
		rc.Logger.Error(err.Error())
		return err
	}

	err = rc.Cluster.ZAdd(ctx, api.KeyForDelayedSending, redis.Z{
		Score:  score,
		Member: emailJSON,
	}).Err()

	duration := time.Since(start).Milliseconds()
	monitoring.RedisLatencyHistogram.Observe(float64(duration))

	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			monitoring.RedisErrorCounter.Inc()
			rc.Logger.Error("AddDelayedEmail: deadline exceeded", zap.Error(err))
			return fmt.Errorf("AddDelayedEmail: %w", context.DeadlineExceeded)

		default:
			monitoring.RedisErrorCounter.Inc()
			rc.Logger.Error("AddDelayedEmail: cannot get entry", zap.Error(err))
			return err
		}
	}

	monitoring.RedisSuccessCounter.Inc()
	return nil
}

func (rc *RedisCluster) CheckRedis(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, rc.Timeout)
	defer cancel()

	start := time.Now()

	now := float64(time.Now().Unix())

	res, err := rc.Cluster.ZRangeByScore(ctx, api.KeyForDelayedSending, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatFloat(now, 'f', -1, 64),
	}).Result()

	duration := time.Since(start).Milliseconds()
	monitoring.RedisLatencyHistogram.Observe(float64(duration))

	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			monitoring.RedisErrorCounter.Inc()
			rc.Logger.Error("CheckRedis: deadline exceeded", zap.Error(err))
			return nil, fmt.Errorf("CheckRedis: %w", context.DeadlineExceeded)

		default:
			monitoring.RedisErrorCounter.Inc()
			rc.Logger.Error("CheckRedis: cannot get entry", zap.Error(err))
			return nil, err
		}
	}

	if len(res) != 0 {
		monitoring.RedisErrorCounter.Inc()
		err = rc.Cluster.ZRem(ctx, api.KeyForDelayedSending, res).Err()
		if err != nil {
			rc.Logger.Warn("CheckRedis: cannot remove entry", zap.Error(err))
		}
	}

	monitoring.RedisSuccessCounter.Inc()
	return res, nil
}

func (rc *RedisCluster) parseAndConvertTime(email *SMTPClient.EmailMessageWithTime) ([]byte, float64, error) {
	UTCTime, err := time.ParseInLocation(emailTimeLayout, email.Time, time.UTC)
	if err != nil {
		rc.Logger.Error("parseAndConvertTime: cannot parse email.Time", zap.Error(err))

		return nil, 0, fmt.Errorf("parseAndConvertTime: cannot parse email.Time: %s: %w", email.Time, err)
	}

	email.Time = strconv.Itoa(int(UTCTime.Unix()))

	jsonEmail, err := json.Marshal(email)
	if err != nil {
		rc.Logger.Error("parseAndConvertTime: failed to marshal email", zap.Error(err))

		return nil, 0, err
	}

	return jsonEmail, float64(UTCTime.Unix()), nil
}
