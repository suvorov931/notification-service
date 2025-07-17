package config

import (
	"fmt"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"notification/internal/SMTPClient"
	"notification/internal/api"
	"notification/internal/logger"
	"notification/internal/storage/postgresClient"
	"notification/internal/storage/redisClient"
)

type Config struct {
	HttpServer  api.HttpServer
	SMTP        SMTPClient.Config
	Redis       redisClient.Config
	Postgres    postgresClient.Config
	Logger      logger.Config
	AppTimeouts AppTimeouts
}

type AppTimeouts struct {
	SMTPPauseForRetries   time.Duration
	SMTPQuantityOfRetries int
	RedisTimeout          time.Duration
	PostgresTimeout       time.Duration
}

func New(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg.AppTimeouts = setAppTimeouts(&cfg)

	return &cfg, nil
}

func setAppTimeouts(cfg *Config) AppTimeouts {
	c := AppTimeouts{}

	if cfg.SMTP.MaxRetries == 0 {
		c.SMTPQuantityOfRetries = SMTPClient.DefaultMaxRetries
	} else {
		c.SMTPQuantityOfRetries = cfg.SMTP.MaxRetries
	}

	if cfg.SMTP.BasicRetryPause == 0 {
		c.SMTPPauseForRetries = SMTPClient.DefaultBasicRetryPause
	} else {
		c.SMTPPauseForRetries = cfg.SMTP.BasicRetryPause
	}

	if cfg.Redis.Timeout == 0 {
		c.RedisTimeout = redisClient.DefaultRedisTimeout
	} else {
		c.RedisTimeout = cfg.Redis.Timeout
	}

	if cfg.Postgres.Timeout == 0 {
		c.PostgresTimeout = postgresClient.DefaultPostgresTimeout
	} else {
		c.PostgresTimeout = cfg.Postgres.Timeout
	}

	return c
}
