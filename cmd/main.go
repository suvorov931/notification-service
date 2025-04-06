package main

import (
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"notification/internal/service"
	"notification/pkg/api"
	"notification/pkg/logger"
)

func main() {
	//	TODO: proto file
	//	TODO: logger
	//	TODO: graceful shutdown
	//	TODO: goroutines?

	logger := logger.New()

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	srv := service.New()
	server := grpc.NewServer()
	api.RegisterNotificationServiceServer(server, srv)

	//go func() {
	logger.Info(fmt.Sprintf("listening on port 50051"))
	if err := server.Serve(lis); err != nil {
		logger.Fatal("failed to serve", zap.Error(err))
	}
	//}()

}
