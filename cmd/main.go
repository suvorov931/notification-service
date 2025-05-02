package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"notification/internal/api"
	"notification/internal/config"
	"notification/internal/logger"
	"notification/internal/service"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	l := logger.New()
	defer l.Sync()

	cfg, err := config.New()
	if err != nil {
		l.Fatal("failed to read config", zap.Error(err))
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", cfg.NotificationsGRPCPort))
	if err != nil {
		l.Fatal("failed to listen", zap.Error(err))
	}

	srv := service.New(cfg, l)
	interceptor := grpc.UnaryInterceptor(logger.Interceptor(l))
	server := grpc.NewServer(interceptor)

	api.RegisterNotificationServiceServer(server, srv)

	go func() {
		if err := server.Serve(lis); err != nil {
			l.Fatal("failed to serve", zap.Error(err))
		}
	}()

	l.Info(fmt.Sprintf("server started"))

	select {
	case <-ctx.Done():
		server.GracefulStop()
		l.Info("server stopped")
	}
}
