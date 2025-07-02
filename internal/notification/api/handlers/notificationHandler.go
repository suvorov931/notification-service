package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"

	"notification/internal/monitoring"
	"notification/internal/notification/SMTPClient"
	"notification/internal/notification/api"
	"notification/internal/notification/api/decoder"
)

func NewSendNotificationHandler(l *zap.Logger, sender SMTPClient.EmailSender, metrics monitoring.Monitoring) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		handlerNameForMetrics := "SendNotification"

		if ctx.Err() != nil {
			metrics.Inc(handlerNameForMetrics, monitoring.StatusCanceled)
			l.Warn("NewSendNotificationHandler: Context canceled before processing started", zap.Error(ctx.Err()))
			return
		}

		start := time.Now()

		email, err := decoder.DecodeEmailRequest(api.KeyForInstantSending, w, r, l)
		if err != nil {
			metrics.Inc(handlerNameForMetrics, monitoring.StatusError)
			l.Error("NewSendNotificationHandler: Failed to decode request", zap.Error(err))
			return
		}

		//_, err = w.Write([]byte("Message is correct,\nStarting to send notification\n\n"))
		//if err != nil {
		//	l.Warn("NewSendNotificationHandler: Cannot send report to caller", zap.Error(err))
		//}
		//
		//if flusher, ok := w.(http.Flusher); ok {
		//	flusher.Flush()
		//} else {
		//	l.Warn("NewSendNotificationHandler: ResponseWriter does not support flushing")
		//}

		err = sender.SendEmail(ctx, *email.(*SMTPClient.EmailMessage))
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				metrics.Inc(handlerNameForMetrics, monitoring.StatusCanceled)
				l.Warn("NewSendNotificationHandler: Request canceled during sending", zap.Error(err))
				return
			}

			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			metrics.Inc(handlerNameForMetrics, monitoring.StatusError)
			l.Error("NewSendNotificationHandler: Cannot send notification", zap.Error(err))
			return
		}

		if _, err = w.Write([]byte("Successfully sent notification\n")); err != nil {
			l.Warn("NewSendNotificationHandler: Cannot send report to caller", zap.Error(err))
		}

		duration := time.Since(start).Seconds()
		metrics.Observe(handlerNameForMetrics, duration)

		metrics.Inc(handlerNameForMetrics, monitoring.StatusSuccess)
	}
}
