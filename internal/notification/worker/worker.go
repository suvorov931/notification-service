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

func Worker(ctx context.Context, rds *redis.Client) []service.Email {
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

func TestData(ctx context.Context, l *zap.Logger, rds *redis.Client) {
	a := service.Email{To: "1",
		Subject: "1",
		Message: "1"}
	aa, _ := json.Marshal(&a)
	err := rds.ZAdd(ctx, "delayed-send", redis.Z{
		Score:  1748845824,
		Member: aa,
	}).Err()
	if err != nil {
		l.Error(err.Error())
	}
	a1 := service.Email{
		To:      "11",
		Subject: "11",
		Message: "11",
	}
	aa1, _ := json.Marshal(&a1)
	err = rds.ZAdd(ctx, "delayed-send", redis.Z{
		Score:  1748845824,
		Member: aa1,
	}).Err()
	if err != nil {
		l.Error(err.Error())
	}
	b := service.Email{
		To:      "2",
		Subject: "2",
		Message: "2",
	}
	bb, _ := json.Marshal(&b)
	err = rds.ZAdd(ctx, "delayed-send", redis.Z{
		Score:  2064378624,
		Member: bb,
	}).Err()
	if err != nil {
		l.Error(err.Error())
	}
	c := service.Email{
		To:      "3",
		Subject: "3",
		Message: "3",
	}
	cc, _ := json.Marshal(&c)
	err = rds.ZAdd(ctx, "delayed-send", redis.Z{
		Score:  2379997824,
		Member: cc,
	}).Err()
	if err != nil {
		l.Error(err.Error())
	}
	d := service.Email{
		To:      "4",
		Subject: "4",
		Message: "4",
	}
	dd, _ := json.Marshal(&d)
	err = rds.ZAdd(ctx, "delayed-send", redis.Z{
		Score:  2695530624,
		Member: dd,
	}).Err()
	if err != nil {
		l.Error(err.Error())
	}
}
