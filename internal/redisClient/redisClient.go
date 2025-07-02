package redisClient

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

func New(ctx context.Context, cfg *Config, metrics monitoring.Monitoring, logger *zap.Logger) (*RedisCluster, error) {
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
		cluster: cluster,
		metrics: metrics,
		logger:  logger,
		timeout: cfg.Timeout,
	}, nil
}

func (rc *RedisCluster) AddDelayedEmail(ctx context.Context, email *SMTPClient.EmailMessageWithTime) error {
	ctx, cancel := context.WithTimeout(ctx, rc.timeout)
	defer cancel()

	emailJSON, score, err := rc.parseAndConvertTime(email)
	if err != nil {
		rc.logger.Error(err.Error())
		return err
	}

	start := time.Now()

	err = rc.cluster.ZAdd(ctx, api.KeyForDelayedSending, redis.Z{
		Score:  score,
		Member: emailJSON,
	}).Err()

	duration := time.Since(start).Seconds()
	rc.metrics.Observe("ZAdd", duration)

	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			rc.metrics.Inc("ZAdd", monitoring.StatusTimeout)
			rc.logger.Error("AddDelayedEmail: deadline exceeded", zap.Error(err))
			return fmt.Errorf("AddDelayedEmail: %w", context.DeadlineExceeded)

		default:
			rc.metrics.Inc("ZAdd", monitoring.StatusError)
			rc.logger.Error("AddDelayedEmail: cannot get entry", zap.Error(err))
			return err
		}
	}

	rc.metrics.Inc("ZAdd", monitoring.StatusSuccess)

	return nil
}

func (rc *RedisCluster) CheckRedis(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, rc.timeout)
	defer cancel()

	now := float64(time.Now().Unix())

	start := time.Now()

	res, err := rc.cluster.ZRangeByScore(ctx, api.KeyForDelayedSending, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatFloat(now, 'f', -1, 64),
	}).Result()

	duration := time.Since(start).Seconds()
	rc.metrics.Observe("ZRangeByScore", duration)

	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			rc.metrics.Inc("ZRangeByScore", monitoring.StatusTimeout)
			rc.logger.Error("CheckRedis: deadline exceeded", zap.Error(err))
			return nil, fmt.Errorf("CheckRedis: %w", context.DeadlineExceeded)

		default:
			rc.metrics.Inc("ZRangeByScore", monitoring.StatusError)
			rc.logger.Error("CheckRedis: cannot get entry", zap.Error(err))
			return nil, err
		}
	}

	rc.metrics.Inc("ZRangeByScore", monitoring.StatusSuccess)

	if len(res) != 0 {
		start = time.Now()

		err = rc.cluster.ZRem(ctx, api.KeyForDelayedSending, res).Err()

		duration = time.Since(start).Seconds()
		rc.metrics.Observe("ZRem", duration)

		if err != nil {
			rc.metrics.Inc("ZRem", monitoring.StatusError)
			rc.logger.Warn("CheckRedis: cannot remove entry", zap.Error(err))
		} else {
			rc.metrics.Inc("ZRem", monitoring.StatusSuccess)
		}
	}

	return res, nil
}

func (rc *RedisCluster) parseAndConvertTime(email *SMTPClient.EmailMessageWithTime) ([]byte, float64, error) {
	UTCTime, err := time.ParseInLocation(emailTimeLayout, email.Time, time.UTC)
	if err != nil {
		rc.logger.Error("parseAndConvertTime: cannot parse email.Time", zap.Error(err))

		return nil, 0, fmt.Errorf("parseAndConvertTime: cannot parse email.Time: %s: %w", email.Time, err)
	}

	email.Time = strconv.Itoa(int(UTCTime.Unix()))

	jsonEmail, err := json.Marshal(email)
	if err != nil {
		rc.logger.Error("parseAndConvertTime: failed to marshal email", zap.Error(err))

		return nil, 0, err
	}

	return jsonEmail, float64(UTCTime.Unix()), nil
}
