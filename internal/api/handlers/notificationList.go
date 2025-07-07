package handlers

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"

	"notification/internal/monitoring"
	"notification/internal/storage/postgresClient"
)

func NewNotificationListHandler(pc postgresClient.PostgresClient, logger *zap.Logger, metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		//start := time.Now()

		handlerNameForMetrics := "SendNotification"

		if ctx.Err() != nil {
			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.IncCanceled(handlerNameForMetrics)
			logger.Warn("NewSendNotificationHandler: Context canceled before processing started", zap.Error(ctx.Err()))
			return
		}

		fmt.Println(r.Body)
	}
}
