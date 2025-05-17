package service

import (
	"context"

	"go.uber.org/zap"

	"notification/internal/config"
)

const (
	maxRetries      = 3
	basicRetryPause = 0.5
)

type Email struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type EmailService struct {
	config *config.CredentialsSender
	logger *zap.Logger
}

type EmailSender interface {
	SendMessage(ctx context.Context, email Email) error
}

func New(config *config.CredentialsSender, logger *zap.Logger) *EmailService {
	return &EmailService{
		config: config,
		logger: logger,
	}
}
