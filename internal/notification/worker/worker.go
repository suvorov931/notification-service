package worker

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"notification/internal/notification/service"
	"notification/internal/rds"
)

const basicTimePeriodForWorker = 1 * time.Second

func Worker(ctx context.Context, rc *rds.RedisClient, logger *zap.Logger) {
	ticker := time.NewTicker(basicTimePeriodForWorker)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker: context canceled")
			return
		case <-ticker.C:
			entry, err := rc.CheckRedis(ctx)
			if err != nil {
				//return err
			}
			fmt.Println(entry)

		}
	}
}

func parseEntry(ctx context.Context, entry []string, sender service.EmailSender) {
	//for _, r := range entry {
	//	sender.SendMessage(ctx)
	//	service.EmailSender(context.Background(), r)
	//}
}
