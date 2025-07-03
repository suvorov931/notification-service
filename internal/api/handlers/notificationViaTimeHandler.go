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

func NewSendNotificationViaTimeHandler(logger *zap.Logger, rc *redisClient.RedisCluster, metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		handlerNameForMetrics := "SendNotificationViaTime"

		if ctx.Err() != nil {
			metrics.Inc(handlerNameForMetrics, monitoring.StatusCanceled)
			logger.Warn("NewSendNotificationViaTimeHandler: Context canceled before processing started", zap.Error(ctx.Err()))
			return
		}

		start := time.Now()

		email, err := decoder.DecodeEmailRequest(api.KeyForDelayedSending, w, r, logger)
		if err != nil {
			metrics.Inc(handlerNameForMetrics, monitoring.StatusError)
			logger.Error("NewSendNotificationViaTimeHandler: Failed to decode request", zap.Error(err))
			return
		}

		_, err = w.Write([]byte("Message is correct\n\n"))
		if err != nil {
			logger.Warn("NewSendNotificationViaTimeHandler: Cannot send report to caller", zap.Error(err))
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		} else {
			logger.Warn("NewSendNotificationViaTimeHandler: ResponseWriter does not support flushing")
		}

		err = rc.AddDelayedEmail(ctx, email.(*SMTPClient.EmailMessageWithTime))
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				metrics.Inc(handlerNameForMetrics, monitoring.StatusCanceled)
				logger.Warn("NewSendNotificationViaTimeHandler: Request canceled during sending", zap.Error(err))
				return
			}

			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.Inc(handlerNameForMetrics, monitoring.StatusError)
			logger.Error("NewSendNotificationViaTimeHandler: Cannot add email", zap.Error(err))
			return
		}

		if _, err = w.Write([]byte("Successfully saved your mail\n")); err != nil {
			logger.Warn("NewSendNotificationViaTimeHandler: Cannot send report to caller", zap.Error(err))
		}

		duration := time.Since(start).Seconds()
		metrics.Observe(handlerNameForMetrics, duration)

		metrics.Inc(handlerNameForMetrics, monitoring.StatusSuccess)
	}
}
