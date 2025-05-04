package mail

import (
	"go.uber.org/zap"

	"notification/internal/config"
)

type Mail struct {
	To      string
	Subject string
	Message string
}

type MailService struct {
	config *config.Config
	logger *zap.Logger
}

type MailSender interface {
	SendMessage(mail Mail) error
}

func New(config *config.Config, logger *zap.Logger) *MailService {
	return &MailService{
		config: config,
		logger: logger,
	}
}
