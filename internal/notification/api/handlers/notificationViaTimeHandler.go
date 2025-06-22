package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"notification/internal/notification/api"
	"notification/internal/notification/api/decoder"
	"notification/internal/notification/service"
	"notification/internal/rds"
)

// TODO: добавить отмену по контексту

func NewSendNotificationViaTimeHandler(l *zap.Logger, rc *rds.RedisClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		email, err := decoder.DecodeEmailRequest(api.KeyForDelayedSending, w, r, l)
		if err != nil {
			return
		}

		_, err = w.Write([]byte("Message is correct\n\n"))
		if err != nil {
			l.Warn("NewSendNotificationViaTimeHandler: Cannot send report to caller", zap.Error(err))
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		} else {
			l.Warn("NewSendNotificationViaTimeHandler: ResponseWriter does not support flushing")
		}

		if err = rc.AddDelayedEmail(ctx, email.(*service.EmailMessageWithTime)); err != nil {
			return
		}

		if _, err = w.Write([]byte("Successfully saved your mail\n")); err != nil {
			l.Warn("NewSendNotificationViaTimeHandler: Cannot send report to caller", zap.Error(err))
		}
	}
}
