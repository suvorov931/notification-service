package service

import (
	"context"

	"go.uber.org/zap"

	"notification/internal/config"
)

type Mail struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type MailService struct {
	config *config.Config
	logger *zap.Logger
}

type MailSender interface {
	SendMessage(ctx context.Context, mail Mail) error
}

func New(config *config.Config, logger *zap.Logger) *MailService {
	return &MailService{
		config: config,
		logger: logger,
	}
}
