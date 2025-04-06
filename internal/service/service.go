package service

import (
	"context"
	"go.uber.org/zap"
	"notification/pkg/api"
	"notification/pkg/logger"
)

type Service struct {
	api.NotificationServiceServer
}

func New() *Service {
	return &Service{}
}

func (s *Service) SendNotification(ctx context.Context, request *api.SendNotificationRequest) (*api.SendNotificationResponse, error) {
	logger.New().Info("", zap.String("mail", request.Mail), zap.String("text", request.Text))
	return &api.SendNotificationResponse{
		Id: 123,
	}, nil
}
