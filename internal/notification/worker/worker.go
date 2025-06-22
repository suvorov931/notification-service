package worker

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"notification/internal/notification/service"
	"notification/internal/rds"
)

const basicTimePeriodForWorker = 1 * time.Second

type Worker struct {
	logger *zap.Logger
	rc     *rds.RedisClient
	sender service.EmailSender
}

func New(logger *zap.Logger, rc *rds.RedisClient, sender service.EmailSender) *Worker {
	return &Worker{
		logger: logger,
		rc:     rc,
		sender: sender,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	ticker := time.NewTicker(basicTimePeriodForWorker)
	defer ticker.Stop()

	w.logger.Info("Worker: started")
	group.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("Worker: context canceled")
				return ctx.Err()

			case <-ticker.C:
				entries, err := w.rc.CheckRedis(ctx)
				if err != nil {
					w.logger.Error("Worker: failed check redis", zap.Error(err))
					continue
				}

				entriesCopy := append([]string(nil), entries...)

				group.Go(func() error {
					return w.processEntries(ctx, entriesCopy)
				})
			}
		}
	})

	if err := group.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			w.logger.Info("Worker: graceful shutdown completed")
			return nil
		}

		w.logger.Error("Worker: shutting down with error", zap.Error(err))
		return err
	}

	return nil
}

func (w *Worker) processEntries(ctx context.Context, entries []string) error {
	for _, entry := range entries {
		select {
		case <-ctx.Done():
			w.logger.Info("processEntries: context canceled")
			return ctx.Err()

		default:

			var res service.EmailMessageWithTime

			if err := json.Unmarshal([]byte(entry), &res); err != nil {
				w.logger.Error("parseAndSendEntry: failed to unmarshal entry", zap.Error(err), zap.String("entry", entry))
				continue
			}

			if err := w.sender.SendEmail(ctx, res.Email); err != nil {
				w.logger.Error("parseEntry: failed to send message", zap.Error(err), zap.Any("email", res.Email))
				continue
			}
		}
	}

	return nil
}
