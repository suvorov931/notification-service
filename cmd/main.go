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

	l := logger.New()

	lis, err := net.Listen("tcp", "0.0.0.0:50051")
	if err != nil {
		l.Fatal("failed to listen", zap.Error(err))
	}

	srv := service.New()
	server := grpc.NewServer(grpc.UnaryInterceptor(logger.Interceptor(l)))
	api.RegisterNotificationServiceServer(server, srv)

	//go func() {
	l.Info(fmt.Sprintf("listening on port 50051"))
	if err := server.Serve(lis); err != nil {
		l.Fatal("failed to serve", zap.Error(err))
	}
	//}()

}
