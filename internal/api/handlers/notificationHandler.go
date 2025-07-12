package handlers

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"notification/internal/api"
	"notification/internal/api/decoder"
	"notification/internal/monitoring"
)

func (nh *NotificationHandler) NewSendNotificationHandler(metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		start := time.Now()

		handlerName := "SendNotification"

		if nh.checkCtxCanceled(ctx, w, metrics, handlerName) {
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
			metrics.IncError(handlerName)
			nh.logger.Error("NewSendNotificationHandler: Cannot put email in postgres", zap.Error(err))
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)

			return
		}

		nh.writeResponseWithId(w, id, "Successfully sent notification", metrics, handlerName)

		metrics.Observe(handlerName, start)
		metrics.IncSuccess(handlerName)
	}
}
