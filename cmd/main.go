package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"notification/internal/config"
	llogger "notification/internal/logger"
	"notification/internal/notification/api/handlers"
	"notification/internal/notification/service"
	wworker "notification/internal/notification/worker"
	"notification/internal/rds"
)

const (
	pathToConfigFile  = "./config/config.yaml"
	tickTimeForWorker = 1 * time.Second
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	cfg, err := config.New(pathToConfigFile)
	if err != nil {
		log.Fatalf("cannot initialize config: %v", err)
	}

	logger, err := llogger.New(&cfg.Logger)
	if err != nil {
		log.Fatalf("cannot initialize logger: %v", err)
	}
	defer logger.Sync()

	redisClient, err := rds.New(ctx, &cfg.Redis, logger)
	if err != nil {
		logger.Fatal("cannot initialize rds client", zap.Error(err))
	}

	smtpClient := service.New(&cfg.SMTP, logger)

	worker := wworker.New(logger, redisClient, smtpClient, tickTimeForWorker)

	go func() {
		err = worker.Run(ctx)
		if err != nil {
			logger.Error("worker exited with error", zap.Error(err))
		}
	}()

	router := initRouter(logger, &cfg.Logger, smtpClient, redisClient)

	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.HttpServer.Host, cfg.HttpServer.Port),
		Handler: router,
	}

	go func() {
		logger.Info("starting http server", zap.String("addr", srv.Addr))
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("cannot start http server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	logger.Info("received shutdown signal")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer shutdownCancel()

	logger.Info("shutting down http server")
	if err = srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("cannot shutdown http server", zap.Error(err))
	} else {
		logger.Info("http server shutdown gracefully")
	}

	logger.Info("stopping http server", zap.String("addr", srv.Addr))

	logger.Info("application shutdown completed successfully")
}

func initRouter(logger *zap.Logger, cfg *llogger.Config, smtpClient *service.SMTPClient, redisClient *rds.RedisCluster) *chi.Mux {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(llogger.MiddlewareLogger(logger, cfg))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/send-notification", handlers.NewSendNotificationHandler(logger, smtpClient))
	router.Post("/send-notification-via-time", handlers.NewSendNotificationViaTimeHandler(logger, redisClient))

	return router
}

// TODO: покрыть тестами ВСЁ
// TODO: дополнить README, написать документацию
// TODO: добавить в stage сборку прогон тестов

// TODO: GitLab CI/CD
// TODO: попробовать на аккаунте лицея
// TODO: github actions

// TODO: мониторинг в целом и редиса, nginx, kafka
// TODO: многопоточность
// TODO: добавить третий хендлер для множественной отправки единого сообщения на разные адреса?
