package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"

	"notification/internal/logger"
	"notification/internal/notification/api"
	"notification/internal/notification/service"
	"notification/internal/rds"
)

type Config struct {
	HttpServer api.HttpServer     `yaml:"HTTP_SERVER" env:"HTTP_SERVER"`
	MailSender service.MailSender `yaml:"MAIL_SENDER" env:"MAIL_SENDER"`
	Redis      rds.Config         `yaml:"REDIS" env:"REDIS"`
	Logger     logger.Config      `yaml:"LOGGER" env:"LOGGER"`
}

func New() (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig("./config/config.yaml", &cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}
