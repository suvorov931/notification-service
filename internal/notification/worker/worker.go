package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"notification/internal/notification/service"
)

const basicTimePeriodForWorker = 1 * time.Second

func Worker(ctx context.Context, logger *zap.Logger) {
	ticker := time.NewTicker(basicTimePeriodForWorker)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker: context canceled")
			return
		case <-ticker.C:
			CheckRedis
		}
	}
}

func orker(ctx context.Context, rds *redis.Client) []service.Email {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			res, err := rds.ZRangeByScore(ctx, "delayed-send", &redis.ZRangeBy{
				Min: "0",
				Max: strconv.Itoa(int(time.Now().Unix() + 10)),
			}).Result()
			if err != nil {
				fmt.Println(err.Error())
			}
			for _, r := range res {
				fmt.Println(r)
			}
			var result []service.Email
			for _, v := range res {
				var r service.Email
				err = json.Unmarshal([]byte(v), &r)
				if err != nil {
					fmt.Println(err.Error())
				}
				result = append(result, r)
			}
			if len(res) != 0 {
				if err = rds.ZRem(ctx, "delayed-send", res).Err(); err != nil {
					fmt.Println(err)
				}
			}
			fmt.Println("end")
			return result
		}
	}
}
