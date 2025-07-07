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

	"notification/internal/SMTPClient"
	"notification/internal/api"
	"notification/internal/monitoring"
)

func New(ctx context.Context, config *Config, metrics monitoring.Monitoring, logger *zap.Logger) (*RedisCluster, error) {
	if config.Timeout == 0 {
		config.Timeout = DefaultRedisTimeout
	}

	pingCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	cluster := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    config.Addrs,
		Password: config.Password,
		ReadOnly: config.ReadOnly,
	})

	if err := cluster.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis cluster: %w", err)
	}

	return &RedisCluster{
		cluster: cluster,
		metrics: metrics,
		logger:  logger,
		timeout: config.Timeout,
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

	rc.metrics.Observe("ZAdd", start)

	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			rc.metrics.IncTimeout("ZAdd")
			rc.logger.Error("AddDelayedEmail: deadline exceeded", zap.Error(err))
			return fmt.Errorf("AddDelayedEmail: %w", context.DeadlineExceeded)

		default:
			rc.metrics.IncError("ZAdd")
			rc.logger.Error("AddDelayedEmail: cannot set entry", zap.Error(err))
			return err
		}
	}

	rc.metrics.IncSuccess("ZAdd")

	rc.metrics.Observe("AddDelayedEmail", start)
	rc.metrics.IncSuccess("AddDelayedEmail")
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

	rc.metrics.Observe("ZRangeByScore", start)

	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			rc.metrics.IncTimeout("ZRangeByScore")
			rc.logger.Error("CheckRedis: deadline exceeded", zap.Error(err))
			return nil, fmt.Errorf("CheckRedis: %w", context.DeadlineExceeded)

		default:
			rc.metrics.IncError("ZRangeByScore")
			rc.logger.Error("CheckRedis: cannot get entry", zap.Error(err))
			return nil, err
		}
	}

	rc.metrics.IncSuccess("ZRangeByScore")

	if len(res) != 0 {
		startZRem := time.Now()

		err = rc.cluster.ZRem(ctx, api.KeyForDelayedSending, res).Err()

		rc.metrics.Observe("ZRem", startZRem)

		if err != nil {
			rc.metrics.IncError("ZRem")
			rc.logger.Warn("CheckRedis: cannot remove entry", zap.Error(err))
		} else {
			rc.metrics.IncSuccess("ZRem")
		}
	}

	rc.metrics.Observe("CheckRedis", start)
	rc.metrics.IncSuccess("CheckRedis")
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
