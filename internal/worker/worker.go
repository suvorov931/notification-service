package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"notification/internal/SMTPClient"
	"notification/internal/monitoring"
	"notification/internal/storage/redisClient"
)

// Worker periodically polls Redis for scheduled email entries and sends them using an SMTP client.
type Worker struct {
	rc           redisClient.RedisClient
	sender       SMTPClient.EmailSender
	metrics      monitoring.Monitoring
	logger       *zap.Logger
	tickDuration time.Duration
}

// New creates and returns a new Worker instance.
func New(rc redisClient.RedisClient, sender SMTPClient.EmailSender, tickDuration time.Duration, metrics monitoring.Monitoring, logger *zap.Logger) *Worker {
	return &Worker{
		rc:           rc,
		sender:       sender,
		tickDuration: tickDuration,
		metrics:      metrics,
		logger:       logger,
	}
}

// Run starts the worker loop, which checks Redis at a configured interval and tries to find scheduled email entries.
// If entries are found, they are processed and emails are sent asynchronously.
func (w *Worker) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	ticker := time.NewTicker(w.tickDuration)
	defer ticker.Stop()

	w.logger.Info("Worker: started")
	group.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				w.metrics.IncCanceled("Worker")
				w.logger.Info("Worker: context canceled")
				return ctx.Err()

			case <-ticker.C:

				entries, err := w.rc.CheckRedis(ctx)
				if err != nil {
					w.metrics.IncError("Worker")
					w.logger.Error("Worker: failed check redis", zap.Error(err))
					continue
				}

				if len(entries) == 0 {
					continue
				}

				entriesCopy := append([]string(nil), entries...)

				w.logger.Info("Worker: got entries from redis", zap.Strings("entries", entriesCopy))

				group.Go(func() error {
					start := time.Now()

					err = w.processEntries(ctx, entriesCopy)

					if err != nil {
						w.metrics.IncError("Worker")
						w.logger.Error("Worker: failed process entries", zap.Error(err))
						return err
					}

					w.metrics.IncSuccess("Worker")
					w.metrics.Observe("Worker", start)
					return nil
				})
			}
		}
	})

	if err := group.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			w.metrics.IncCanceled("Worker")
			w.logger.Info("Worker: graceful shutdown completed")
			return nil
		}

		w.metrics.IncError("Worker")
		w.logger.Error("Worker: shutting down with error", zap.Error(err))
		return err
	}

	return nil
}

// processEntries handles a batch of entries received from Redis.
// It decodes each entry and sends the corresponding email using the SMTP client.
func (w *Worker) processEntries(ctx context.Context, entries []string) error {
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			w.metrics.IncCanceled("Worker")
			w.logger.Info("processEntries: context canceled")
			return ctx.Err()

		default:

			var email SMTPClient.TempEmailMessage

			if err := json.Unmarshal([]byte(entry), &email); err != nil {
				w.metrics.IncError("Worker")
				w.logger.Error("processEntries: failed to unmarshal entry", zap.Error(err), zap.String("entry", entry))
				continue
			}

			res := SMTPClient.EmailMessage{
				To:      email.To,
				Subject: email.Subject,
				Message: email.Message,
			}

			if err := w.sender.SendEmail(ctx, res); err != nil {
				w.metrics.IncError("Worker")
				w.logger.Error("processEntries: failed to send message", zap.Error(err), zap.Any("email", email))
				continue
			}

			w.logger.Info("Worker: successfully sent delayed message", zap.Any("email", email))
		}
	}

	return nil
}
