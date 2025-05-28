package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"notification/internal/notification/api/decoder"
	"notification/internal/notification/service"
)

func NewSendNotificationViaTimeHandler(l *zap.Logger, sender service.EmailSender, rds *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//ctx := r.Context()

		email, err := decoder.DecodeEmailRequest(decoder.KeyForDelayedSending, w, r, l)
		if err != nil {
			return
		}

		ctx := context.Background()
		emailJSON, s, err := parseAndConvertTime(l, email.(*service.EmailWithTime))
		if err != nil {
			l.Error(err.Error())
		}
		fmt.Println(s)
		err = rds.ZAdd(ctx, "delayed-send", redis.Z{
			Score:  s,
			Member: emailJSON,
		}).Err()
		if err != nil {
			l.Error(err.Error())
		}
	}
}

func parseAndConvertTime(l *zap.Logger, email *service.EmailWithTime) ([]byte, float64, error) {
	UTCTime, err := time.ParseInLocation("2006-01-02 15:04:05", email.Time, time.UTC)
	if err != nil {
		l.Error("cannot parse email.Time", zap.Error(err))
		return nil, 0, err
	}

	email.Time = strconv.Itoa(int(UTCTime.Unix()))

	jsonEmail, err := json.Marshal(email)
	if err != nil {
		l.Error("ParseAndConvertTime: failed to marshal email", zap.Error(err))
		return nil, 0, err
	}

	return jsonEmail, float64(UTCTime.Unix()), nil
}
