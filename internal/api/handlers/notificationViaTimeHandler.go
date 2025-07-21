package handlers

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"

	"notification/internal/api"
	"notification/internal/api/decoder"
	"notification/internal/monitoring"
)

// NewSendNotificationViaTimeHandler returns an HTTP handler that handles delayed email notifications.
// It decodes and validates the request, stores email to Redis,
// saves the message to PostgreSQL, and writes a response on success.
func (nh *NotificationHandler) NewSendNotificationViaTimeHandler(metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), nh.calculateTimeoutForSendViaTime())
		defer cancel()

		start := time.Now()

		handlerName := "SendNotificationViaTime"

		if nh.checkCtxError(ctx, w, metrics, handlerName) {
			return
		}

		email, err := decoder.DecodeRequest(nh.logger, r, w, api.KeyForDelayedSending)
		if err != nil {
			metrics.IncError(handlerName)
			nh.logger.Error("NewSendNotificationViaTimeHandler: Failed to decode request", zap.Error(err))
			return
		}

		err = nh.redisClient.AddDelayedEmail(ctx, email)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.IncError(handlerName)
			nh.logger.Error("NewSendNotificationViaTimeHandler: Cannot add entry", zap.Error(err))

			return
		}

		id, err := nh.postgresClient.SaveEmail(ctx, email)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.IncError(handlerName)
			nh.logger.Error("NewSendNotificationViaTimeHandler: Cannot put email in postgres", zap.Error(err))

			return
		}

		nh.writeResponseWithId(w, id, "Successfully saved your mail", metrics, handlerName)

		metrics.Observe(handlerName, start)
		metrics.IncSuccess(handlerName)
	}
}
