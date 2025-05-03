package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"

	"notification/internal/logger"
	"notification/internal/rds"
)

type Config struct {
	HttpServer        HttpServer        `yaml:"HTTP_SERVER" env:"HTTP_SERVER"`
	CredentialsSender CredentialsSender `yaml:"CREDENTIALS_SENDER" env:"CREDENTIALS_SENDER"`
	Redis             rds.Config        `yaml:"REDIS" env:"REDIS"`
	Logger            logger.Config     `yaml:"LOGGER" env:"LOGGER"`
}

type HttpServer struct {
	Addr string `yaml:"HTTP_ADDR" env:"HTTP_ADDR"`
}

type CredentialsSender struct {
	SenderEmail    string `yaml:"SENDER_EMAIL" env:"SENDER_EMAIL"`
	SenderPassword string `yaml:"SENDER_PASSWORD" env:"SENDER_PASSWORD"`
	SMTPHost       string `yaml:"SMTP_HOST" env:"SMTP_HOST"`
	SMTPPORT       int    `yaml:"SMTP_PORT" env:"SMTP_PORT"`
}

func New() (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig("./config/config.yaml", &cfg); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	return &cfg, nil
}
