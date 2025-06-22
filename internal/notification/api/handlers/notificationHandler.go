package handlers

import (
	"context"
	"errors"
	"net/http"

	"go.uber.org/zap"

	"notification/internal/notification/api"
	"notification/internal/notification/api/decoder"
	"notification/internal/notification/service"
)

func NewSendNotificationHandler(l *zap.Logger, sender service.EmailSender) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		email, err := decoder.DecodeEmailRequest(api.KeyForInstantSending, w, r, l)
		if err != nil {
			return
		}

		_, err = w.Write([]byte("Message is correct,\nStarting to send notification\n\n"))
		if err != nil {
			l.Warn("NewSendNotificationHandler: Cannot send report to caller", zap.Error(err))
		}

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		} else {
			l.Warn("NewSendNotificationHandler: ResponseWriter does not support flushing")
		}

		err = sender.SendEmail(ctx, *email.(*service.EmailMessage))
		if err != nil {
			if errors.Is(ctx.Err(), context.Canceled) {
				l.Warn("NewSendNotificationHandler: Request canceled during sending", zap.Error(err))
				return
			}

			http.Error(w, http.StatusText(500), http.StatusInternalServerError)
			l.Error("NewSendNotificationHandler: Cannot send notification", zap.Error(err))
			return
		}

		if _, err = w.Write([]byte("Successfully sent notification\n")); err != nil {
			l.Warn("NewSendNotificationHandler: Cannot send report to caller", zap.Error(err))
		}
	}

}
