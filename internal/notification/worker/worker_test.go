package worker

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"notification/internal/logger"
	"notification/internal/notification/api"
	"notification/internal/notification/service"
	"notification/internal/rds"
)

// TODO: красиво обернуть, залогировать (отправляю сообщение... в воркере), обернуть ошибки
// TODO: в воркере хапать текущее время и на 1 секунду больше \
// TODO: (min: time.Now().Unix(), max: time.Now().Add(1 * time.Second).Unix())

func TestWorker(t *testing.T) {
	ctx := context.Background()
	l, _ := logger.New(&logger.Config{Env: "dev"})
	rc, _ := rds.New(ctx, &rds.Config{Addr: "localhost:6379", Password: "12345"}, zap.NewNop())

	rc.Client.Del(ctx, api.KeyForDelayedSending)

	now := int(time.Now().Add(2 * time.Second).Unix())
	e := &service.EmailWithTime{
		Time: strconv.Itoa(now),
		Email: service.Email{
			To:      "1",
			Subject: "2",
			Message: "3",
		},
	}

	jsonEmail, err := json.Marshal(e)
	if err != nil {
		t.Error(err)
	}

	err = rc.Client.ZAdd(ctx, api.KeyForDelayedSending, redis.Z{
		Score:  float64(now),
		Member: jsonEmail,
	}).Err()
	if err != nil {
		t.Fatal(err)
	}

	//res, err := rc.Client.ZRangeByScore(ctx, api.KeyForDelayedSending, &redis.ZRangeBy{
	//	Min: strconv.Itoa(int(time.Now().Unix())),
	//	Max: strconv.Itoa(int(time.Now().Unix())),
	//}).Result()
	//
	//fmt.Println(res)

	Worker(ctx, rc, l)

	//time.Sleep(10 * time.Second)
}
