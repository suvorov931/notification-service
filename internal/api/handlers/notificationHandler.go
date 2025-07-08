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

func NewSendNotificationHandler(sender SMTPClient.EmailSender, pc postgresClient.PostgresClient,
	logger *zap.Logger, metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		start := time.Now()

		handlerNameForMetrics := "SendNotification"

		if ctx.Err() != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.IncCanceled(handlerNameForMetrics)
			logger.Warn("NewSendNotificationHandler: Context canceled before processing started", zap.Error(ctx.Err()))
			return
		}

		rawEmail, err := decoder.DecodeEmailRequest(api.KeyForInstantSending, w, r, logger)
		if err != nil {
			metrics.IncError(handlerNameForMetrics)
			logger.Error("NewSendNotificationHandler: Failed to decode request", zap.Error(err))
			return
		}

		email := rawEmail.(*SMTPClient.EmailMessage)

		err = sender.SendEmail(ctx, *email)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				metrics.IncCanceled("SendNotification")
				logger.Warn("NewSendNotificationHandler: Request canceled during sending", zap.Error(err))
			} else {
				metrics.IncError(handlerNameForMetrics)
				logger.Error("NewSendNotificationHandler: Cannot send notification", zap.Error(err))
			}

			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		id, err := pc.SavingInstantSending(ctx, email)
		if err != nil {
			metrics.IncError(handlerNameForMetrics)
			logger.Warn("NewSendNotificationHandler: Cannot put email in postgres")
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			return
		}

		if err = writeResponse(w, id, "Successfully sent notification"); err != nil {
			metrics.IncError(handlerNameForMetrics)
			logger.Error("NewSendNotificationHandler: Cannot send report to caller", zap.Error(err))
		}

		metrics.Observe(handlerNameForMetrics, start)

		metrics.IncSuccess(handlerNameForMetrics)
	}
}
