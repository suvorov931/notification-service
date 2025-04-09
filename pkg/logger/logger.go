package logger

import (
	"context"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"time"
)

func New() *zap.Logger {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	return logger
}

func Interceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		next grpc.UnaryHandler,
	) (resp any, err error) {

		logger.Info(
			"new request", zap.String("method", info.FullMethod),
			zap.Any("request", req),
			zap.Time("time", time.Now()),
		)

		return next(ctx, req)
	}
}
