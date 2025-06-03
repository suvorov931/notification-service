package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"notification/internal/notification/api"
	"notification/internal/notification/api/decoder"
	rds2 "notification/internal/rds"
)

func NewSendNotificationViaTimeHandler(l *zap.Logger, rc rds2.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		email, err := decoder.DecodeEmailRequest(api.KeyForDelayedSending, w, r, l)
		if err != nil {
			return
		}

		_, err = w.Write([]byte("Message is correct\n"))
		if err != nil {
			l.Warn("NewSendNotificationViaTimeHandler: Cannot send report to caller", zap.Error(err))
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		} else {
			l.Warn("NewSendNotificationViaTimeHandler: ResponseWriter does not support flushing")
		}

		rc.AddDelayedEmail(ctx, email)

	}
}
