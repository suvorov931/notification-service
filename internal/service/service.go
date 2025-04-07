package service

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"notification/pkg/api"
)

type Service struct {
	api.NotificationServiceServer
}

func New() *Service {
	return &Service{}
}

func (s *Service) SendNotification(ctx context.Context, request *api.SendNotificationRequest) (*api.SendNotificationResponse, error) {
	id := uuid.New().String()
	fmt.Println(id)

	return &api.SendNotificationResponse{
		Id: 123,
	}, nil
}
