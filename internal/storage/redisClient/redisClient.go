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

// New creates and returns a new RedisCluster instance, applies default timeout if not set.
func New(ctx context.Context, config *Config, metrics monitoring.Monitoring, logger *zap.Logger) (*RedisCluster, error) {
	if config.Timeout == 0 {
		config.Timeout = DefaultRedisTimeout
	}

	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	cluster := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:    config.Addrs,
		Password: config.Password,
		ReadOnly: config.ReadOnly,
	})

	if err := cluster.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis cluster: %w", err)
	}

	return &RedisCluster{
		cluster:         cluster,
		metrics:         metrics,
		logger:          logger,
		timeout:         config.Timeout,
		shutdownTimeout: config.ShutdownTimeout,
	}, nil
}

// AddDelayedEmail adds an email to a Redis sorted set,
// using the email's UNIX timestamp as the score and the serialized email as the member.
func (rc *RedisCluster) AddDelayedEmail(ctx context.Context, email *SMTPClient.EmailMessage) error {
	ctx, cancel := context.WithTimeout(ctx, rc.timeout)
	defer cancel()

	start := time.Now()

	emailJSON, score, err := rc.parseAndConvertData(email)
	if err != nil {
		rc.metrics.IncError("AddDelayedEmail")
		rc.logger.Error("AddDelayedEmail: cannot parse email.Time", zap.Error(err))
		return err
	}

	err = rc.cluster.ZAdd(ctx, api.KeyForDelayedSending, redis.Z{
		Score:  score,
		Member: emailJSON,
	}).Err()
	if err != nil {
		return rc.processContextError("AddDelayedEmail", err)
	}

	rc.metrics.Observe("AddDelayedEmail", start)
	rc.metrics.IncSuccess("AddDelayedEmail")

	return nil
}

// CheckRedis retrieves all delayed emails whose scheduled time has passed,
// removes them from the Z-Set, and returns them as a list of JSON strings.
func (rc *RedisCluster) CheckRedis(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, rc.timeout)
	defer cancel()

	start := time.Now()

	now := float64(time.Now().Unix())

	res, err := rc.cluster.ZRangeByScore(ctx, api.KeyForDelayedSending, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatFloat(now, 'f', -1, 64),
	}).Result()
	if err != nil {
		return nil, rc.processContextError("CheckRedis", err)
	}

	if len(res) != 0 {
		err = rc.cluster.ZRem(ctx, api.KeyForDelayedSending, res).Err()

		if err != nil {
			rc.metrics.IncError("CheckRedis")
			rc.logger.Warn("CheckRedis: cannot remove entry", zap.Error(err))
		}
	}

	rc.metrics.Observe("CheckRedis", start)
	rc.metrics.IncSuccess("CheckRedis")

	return res, nil
}

// Close shuts down all Redis Cluster nodes.
func (rc *RedisCluster) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), rc.shutdownTimeout)
	defer cancel()

	start := time.Now()

	done := make(chan error, 1)
	go func() {
		done <- rc.cluster.Close()
	}()

	select {
	case <-ctx.Done():
		rc.metrics.IncTimeout("Close")
		rc.logger.Error("Close: timeout closing redis cluster")
		return fmt.Errorf("Close: timeout closing redis cluster: %w", ctx.Err())

	case err := <-done:
		if err != nil {
			rc.metrics.IncError("Close")
			rc.logger.Error("Close: cannot close cluster", zap.Error(err))
			return err
		}

		rc.metrics.Observe("Close", start)
		rc.metrics.IncSuccess("Close")
		return nil
	}
}

// parseAndConvertData serializes the email to JSON and converts its time to a UNIX timestamp score.
func (rc *RedisCluster) parseAndConvertData(email *SMTPClient.EmailMessage) ([]byte, float64, error) {
	unixTime := email.Time.Unix()

	t := strconv.FormatInt(unixTime, 10)

	jsonStruct := SMTPClient.TempEmailMessage{
		Type:    email.Type,
		Time:    t,
		To:      email.To,
		Subject: email.Subject,
		Message: email.Message,
	}

	jsonEmail, err := json.Marshal(jsonStruct)
	if err != nil {
		rc.logger.Error("parseAndConvertEmail: failed to marshal email", zap.Error(err))
		return nil, 0, fmt.Errorf("parseAndConvertEmail: failed to marshal email: %w", err)
	}

	return jsonEmail, float64(email.Time.Unix()), nil
}

// processContextError handles and returns wrapped specified error.
func (rc *RedisCluster) processContextError(funcName string, err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		rc.metrics.IncCanceled(funcName)
		rc.logger.Error(fmt.Sprintf("%s: context canceled", funcName), zap.Error(err))

		return fmt.Errorf("%s: context canceled: %w", funcName, err)

	case errors.Is(err, context.DeadlineExceeded):
		rc.metrics.IncTimeout(funcName)
		rc.logger.Error(fmt.Sprintf("%s: deadline context", funcName), zap.Error(err))

		return fmt.Errorf("%s: deadline context: %w", funcName, err)

	default:
		rc.metrics.IncError(funcName)
		rc.logger.Error(funcName, zap.Error(err))

		return fmt.Errorf("%s: %w", funcName, err)
	}
}
