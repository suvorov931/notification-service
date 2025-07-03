package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"notification/internal/SMTPClient"
	"notification/internal/api"
	"notification/internal/api/decoder"
	"notification/internal/monitoring"
	"notification/internal/redisClient"
)

func NewSendNotificationViaTimeHandler(logger *zap.Logger, rc redisClient.RedisClient, metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		start := time.Now()

		handlerNameForMetrics := "SendNotificationViaTime"

		if ctx.Err() != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.Inc(handlerNameForMetrics, monitoring.StatusCanceled)
			logger.Warn("NewSendNotificationViaTimeHandler: Context canceled before processing started", zap.Error(ctx.Err()))
			return
		}

		email, err := decoder.DecodeEmailRequest(api.KeyForDelayedSending, w, r, logger)
		if err != nil {
			metrics.Inc(handlerNameForMetrics, monitoring.StatusError)
			logger.Error("NewSendNotificationViaTimeHandler: Failed to decode request", zap.Error(err))
			return
		}

		err = rc.AddDelayedEmail(ctx, email.(*SMTPClient.EmailMessageWithTime))
		if err != nil {
			var status string

			if errors.Is(err, context.Canceled) {
				status = monitoring.StatusCanceled
				logger.Warn("NewSendNotificationHandler: Request canceled during sending", zap.Error(err))
			} else {
				status = monitoring.StatusError
				logger.Error("NewSendNotificationHandler: Cannot send notification", zap.Error(err))
			}

			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.Inc(handlerNameForMetrics, status)
			return
		}

		if _, err = w.Write([]byte("\nSuccessfully saved your mail\n")); err != nil {
			logger.Warn("NewSendNotificationViaTimeHandler: Cannot send report to caller", zap.Error(err))
		}

		duration := time.Since(start).Seconds()
		metrics.Observe(handlerNameForMetrics, duration)

		metrics.Inc(handlerNameForMetrics, monitoring.StatusSuccess)
	}
}
