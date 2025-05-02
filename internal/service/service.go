package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"

	"notification/internal/api"
	"notification/internal/config"
)

const (
	maxRetries      = 3
	basicRetryPause = 5
)

type Service struct {
	api.NotificationServiceServer
	config *config.Config
	logger *zap.Logger
}

func New(config *config.Config, logger *zap.Logger) *Service {
	return &Service{
		config: config,
		logger: logger,
	}
}

func (s *Service) sendMessage(to string, subject string, message string) error {
	msg := gomail.NewMessage()

	msg.SetHeader("From", s.config.SendMail.SenderEmail)
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/plain", message)

	dialer := gomail.NewDialer(
		"smtp.mail.ru",
		587,
		s.config.SendMail.SenderEmail,
		s.config.SendMail.SenderPassword,
	)

	err := dialer.DialAndSend(msg)
	if err != nil {
		for i := 0; i < maxRetries; i++ {
			pause := math.Pow(2, float64(i)) + basicRetryPause
			time.Sleep(time.Duration(pause) * time.Second)

			err = dialer.DialAndSend(msg)
			if err == nil {
				s.logger.Info("Successfully sent message")
				return nil
			}
		}

		s.logger.Error(cannotSendMessage(to, err))
		return fmt.Errorf(cannotSendMessage(to, err))
	}

	s.logger.Info("Successfully sent message")
	return nil
}

func cannotSendMessage(to string, err error) string {
	return fmt.Sprintf("cannot send message to %s : %v", to, err)
}

func (s *Service) SendNotification(ctx context.Context, request *api.SendNotificationRequest) (*api.SendNotificationResponse, error) {
	err := s.sendMessage(request.Mail, request.Subject, request.Text)
	if err != nil {
		return nil, fmt.Errorf("SendNotification: failed to send message: %w", err)
	}

	id := uuid.New().String()

	return &api.SendNotificationResponse{
		Id: id,
	}, nil
}
