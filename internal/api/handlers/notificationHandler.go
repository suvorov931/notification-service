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

// NewSendNotificationHandler returns an HTTP handler that handles instant email notifications.
// It decodes and validates the request, sends the email,
// saves the message to PostgreSQL, and writes a response on success.
func (nh *NotificationHandler) NewSendNotificationHandler(metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), nh.calculateTimeoutForSend())
		defer cancel()

		start := time.Now()

		handlerName := "SendNotification"

		if nh.checkCtxError(ctx, w, metrics, handlerName) {
			return
		}

		email, err := decoder.DecodeRequest(nh.logger, r, w, api.KeyForInstantSending)
		if err != nil {
			metrics.IncError(handlerName)
			nh.logger.Error("NewSendNotificationHandler: Failed to decode request", zap.Error(err))
			return
		}

		err = nh.sender.SendEmail(ctx, *email)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.IncError(handlerName)
			nh.logger.Error("NewSendNotificationHandler: Cannot send notification", zap.Error(err))

			return
		}

		id, err := nh.postgresClient.SaveEmail(ctx, email)
		if err != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.IncError(handlerName)
			nh.logger.Error("NewSendNotificationHandler: Cannot put email in postgres", zap.Error(err))

			return
		}

		nh.writeResponseWithId(w, id, "Successfully sent notification", metrics, handlerName)

		metrics.Observe(handlerName, start)
		metrics.IncSuccess(handlerName)
	}
}
