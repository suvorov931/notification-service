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
)

func NewSendNotificationHandler(sender SMTPClient.EmailSender, pc postgresClient.PostgresClient, logger *zap.Logger, metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		start := time.Now()

		handlerNameForMetrics := "SendNotification"

		if ctx.Err() != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.Inc(handlerNameForMetrics, monitoring.StatusCanceled)
			logger.Warn("NewSendNotificationHandler: Context canceled before processing started", zap.Error(ctx.Err()))
			return
		}

		email, err := decoder.DecodeEmailRequest(api.KeyForInstantSending, w, r, logger)
		if err != nil {
			metrics.Inc(handlerNameForMetrics, monitoring.StatusError)
			logger.Error("NewSendNotificationHandler: Failed to decode request", zap.Error(err))
			return
		}

		err = sender.SendEmail(ctx, *email.(*SMTPClient.EmailMessage))
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

		if _, err = w.Write([]byte("\nSuccessfully sent notification\n")); err != nil {
			logger.Warn("NewSendNotificationHandler: Cannot send report to caller", zap.Error(err))
		}

		err = pc.AddSending(ctx, api.KeyForInstantSending, email.(*SMTPClient.EmailMessage))
		if err != nil {
			metrics.Inc(handlerNameForMetrics, monitoring.StatusError)
			logger.Warn("NewSendNotificationHandler: Cannot put email in postgres")
		}

		duration := time.Since(start).Seconds()
		metrics.Observe(handlerNameForMetrics, duration)

		metrics.Inc(handlerNameForMetrics, monitoring.StatusSuccess)
	}
}
