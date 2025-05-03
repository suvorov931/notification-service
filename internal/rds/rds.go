package rds

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Config struct {
	Addr     string `yaml:"REDIS_ADDR" env:"REDIS_ADDR"`
	Password string `yaml:"REDIS_PASSWORD" env:"REDIS_PASSWORD"`
	//DB       int    `yaml:"REDIS_DB" env:"REDIS_DB"`
	//Username string `yaml:"REDIS_USERNAME" env:"REDIS_USERNAME"`
}

func New(ctx context.Context, cfg Config) (*redis.Client, error) {
	cl := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		//DB:       cfg.DB,
		//Username: cfg.Username,
	})

	if err := cl.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return cl, nil
}
