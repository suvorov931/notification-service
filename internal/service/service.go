package service

import (
	"context"
	"github.com/google/uuid"
	"notification/pkg/api"
)

type Service struct {
	api.NotificationServiceServer
}

func New() *Service {
	return &Service{}
}

func NewNot() api.Notification {

}

func (s *Service) SendNotification(ctx context.Context, request *api.SendNotificationRequest) (*api.SendNotificationResponse, error) {
	id := uuid.New().String()

	return &api.SendNotificationResponse{
		Id: id,
	}, nil
}
