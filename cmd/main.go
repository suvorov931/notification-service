package main

import (
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"notification/internal/config"
	"notification/internal/service"
	"notification/pkg/api"
	"notification/pkg/logger"
)

func main() {
	//	TODO: graceful shutdown
	//  TODO: timeout and retry
	//	TODO: goroutines?

	l := logger.New()

	cfg, err := config.New()
	if err != nil {
		l.Fatal("failed to read config", zap.Error(err))
	}

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		l.Fatal("failed to listen", zap.Error(err))
	}

	srv := service.New(cfg, l)
	interceptor := grpc.UnaryInterceptor(logger.Interceptor(l))
	s := grpc.NewServer(interceptor)
	api.RegisterNotificationServiceServer(s, srv)

	//go func() {
	l.Info(fmt.Sprintf("listening on port 50051"))
	if err := s.Serve(lis); err != nil {
		l.Fatal("failed to serve", zap.Error(err))
	}
	//}()

	//if err := sendMessage.SendMessage(cfg, l, "daanisimov04@gmail.com", "hi", "hello"); err != nil {
	//	l.Fatal("failed to send message", zap.Error(err))
	//}
}
