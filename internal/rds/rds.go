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
	Client *redis.Client
	Logger *zap.Logger
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

	return &RedisClient{Client: cl, Logger: logger}, nil
}

func (rc *RedisClient) AddDelayedEmail(ctx context.Context, email *service.EmailWithTime) error {
	emailJSON, scr, err := rc.parseAndConvertTime(email)
	if err != nil {
		rc.Logger.Error(err.Error())

		return err
	}

	err = rc.Client.ZAdd(ctx, api.KeyForDelayedSending, redis.Z{
		Score:  scr,
		Member: emailJSON,
	}).Err()
	if err != nil {
		rc.Logger.Error("AddDelayedEmail: cannot save email in redis", zap.Error(err))
	}

	return nil
}

func (rc *RedisClient) CheckRedis(ctx context.Context) ([]string, error) {
	res, err := rc.Client.ZRangeByScore(ctx, api.KeyForDelayedSending, &redis.ZRangeBy{
		Min: strconv.Itoa(int(time.Now().Unix())),
		Max: strconv.Itoa(int(time.Now().Unix())),
	}).Result()
	if err != nil {
		rc.Logger.Error("CheckRedis: cannot get entry", zap.Error(err))
		return nil, err
	}

	if len(res) != 0 {
		err = rc.Client.ZRem(ctx, api.KeyForDelayedSending, res).Err()
		if err != nil {
			rc.Logger.Warn("CheckRedis: cannot remove entry", zap.Error(err))
		}
	}

	return res, nil
}

func (rc *RedisClient) parseAndConvertTime(email *service.EmailWithTime) ([]byte, float64, error) {
	UTCTime, err := time.ParseInLocation("2006-01-02 15:04:05", email.Time, time.Local)
	if err != nil {
		rc.Logger.Error("parseAndConvertTime: cannot parse email.Time", zap.Error(err))

		return nil, 0, err
	}

	email.Time = strconv.Itoa(int(UTCTime.Unix()))

	jsonEmail, err := json.Marshal(email)
	if err != nil {
		rc.Logger.Error("parseAndConvertTime: failed to marshal email", zap.Error(err))

		return nil, 0, err
	}

	return jsonEmail, float64(UTCTime.Unix()), nil
}
