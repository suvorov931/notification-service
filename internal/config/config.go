package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"

	"notification/internal/logger"
	"notification/internal/notification/SMTPClient"
	"notification/internal/notification/api"
	"notification/internal/redisClient"
)

type Config struct {
	HttpServer api.HttpServer
	SMTP       SMTPClient.Config
	Redis      redisClient.Config
	Logger     logger.Config
}

func New(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}
