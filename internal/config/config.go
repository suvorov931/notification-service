package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	GrpcPort       int    `yaml:"GRPC_PORT" env:"GRPC_PORT" env-default:"50051"`
	SenderEmail    string `yaml:"SENDER_EMAIL" env:"SENDER_EMAIL"`
	SenderPassword string `yaml:"SENDER_PASSWORD" env:"SENDER_PASSWORD"`
}

func New() (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig("./config/config.yaml", &cfg); err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	return &cfg, nil
}
