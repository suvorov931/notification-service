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
	"notification/internal/storage/postgresClient"
	"notification/internal/storage/redisClient"
)

func NewSendNotificationViaTimeHandler(rc redisClient.RedisClient, pc postgresClient.PostgresClient,
	logger *zap.Logger, metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		start := time.Now()

		handlerNameForMetrics := "SendNotificationViaTime"

		if ctx.Err() != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.IncCanceled(handlerNameForMetrics)
			logger.Error("NewSendNotificationViaTimeHandler: Context canceled before processing started", zap.Error(ctx.Err()))
			return
		}

		rawEmail, err := decoder.DecodeEmailRequest(api.KeyForDelayedSending, w, r, logger)
		if err != nil {
			metrics.IncError(handlerNameForMetrics)
			logger.Error("NewSendNotificationViaTimeHandler: Failed to decode request", zap.Error(err))
			return
		}

		email := rawEmail.(*SMTPClient.EmailMessageWithTime)

		err = rc.AddDelayedEmail(ctx, email)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				metrics.IncCanceled(handlerNameForMetrics)
				logger.Error("NewSendNotificationViaTimeHandler: Request canceled during sending", zap.Error(err))
			} else {
				metrics.IncError(handlerNameForMetrics)
				logger.Error("NewSendNotificationViaTimeHandler: Cannot send notification", zap.Error(err))
			}

			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		id, err := pc.SaveEmail(ctx, email)
		if err != nil {
			metrics.IncError(handlerNameForMetrics)
			logger.Error("NewSendNotificationViaTimeHandler: Cannot put email in postgres")
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		if err = writeResponseWithId(w, id, "Successfully saved your mail"); err != nil {
			metrics.IncError(handlerNameForMetrics)
			logger.Error("NewSendNotificationViaTimeHandler: Cannot send report to caller", zap.Error(err))
		}

		metrics.Observe(handlerNameForMetrics, start)

		metrics.IncSuccess(handlerNameForMetrics)
	}
}
