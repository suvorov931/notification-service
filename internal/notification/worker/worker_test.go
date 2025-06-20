package worker

import (
	"context"
	"testing"

	"notification/internal/logger"
)

func TestWorker(t *testing.T) {
	ctx := context.Background()
	l, _ := logger.New(&logger.Config{Env: "dev"})
	Worker(ctx, l)
}
