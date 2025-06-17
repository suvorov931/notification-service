package rds

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"notification/internal/notification/api"
	"notification/internal/notification/service"
)

// TODO: сделать проверку на то, что сообщение уже сохранялось ранее

type Config struct {
	Addr     string `yaml:"REDIS_ADDR" env:"REDIS_ADDR"`
	Password string `yaml:"REDIS_PASSWORD" env:"REDIS_PASSWORD"`
	//DB       int    `yaml:"REDIS_DB" env:"REDIS_DB"`
	//Username string `yaml:"REDIS_USERNAME" env:"REDIS_USERNAME"`
}

type RedisClient struct {
	client *redis.Client
	logger *zap.Logger
}

func New(ctx context.Context, cfg *Config, logger *zap.Logger) (*RedisClient, error) {
	cl := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		//DB:       cfg.DB,
		//Username: cfg.Username,
	})

	if err := cl.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &RedisClient{client: cl, logger: logger}, nil
}

func (rc *RedisClient) AddDelayedEmail(ctx context.Context, email *service.EmailWithTime) error {
	emailJSON, scr, err := rc.parseAndConvertTime(email)
	if err != nil {
		rc.logger.Error(err.Error())

		return err
	}

	err = rc.client.ZAdd(ctx, api.KeyForDelayedSending, redis.Z{
		Score:  scr,
		Member: emailJSON,
	}).Err()
	if err != nil {
		rc.logger.Error("AddDelayedEmail: cannot save email in redis", zap.Error(err))
	}

	return nil
}

func (rc *RedisClient) parseAndConvertTime(email *service.EmailWithTime) ([]byte, float64, error) {
	UTCTime, err := time.ParseInLocation("2006-01-02 15:04:05", email.Time, time.UTC)
	if err != nil {
		rc.logger.Error("parseAndConvertTime: cannot parse email.Time", zap.Error(err))

		return nil, 0, err
	}

	email.Time = strconv.Itoa(int(UTCTime.Unix()))

	jsonEmail, err := json.Marshal(email)
	if err != nil {
		rc.logger.Error("parseAndConvertTime: failed to marshal email", zap.Error(err))

		return nil, 0, err
	}

	return jsonEmail, float64(UTCTime.Unix()), nil
}
