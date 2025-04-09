package main

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"notification/internal/config"
	"notification/internal/service"
	"notification/pkg/api"
	"notification/pkg/logger"
	"os"
	"os/signal"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

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

	go func() {
		if err := s.Serve(lis); err != nil {
			l.Fatal("failed to serve", zap.Error(err))
		}
	}()

	l.Info(fmt.Sprintf("server started"))

	select {
	case <-ctx.Done():
		s.GracefulStop()
		l.Info("server stopped")
	}
}
