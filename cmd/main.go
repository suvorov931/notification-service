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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"

	cconfig "notification/internal/config"
	llogger "notification/internal/logger"
	"notification/internal/monitoring"
	"notification/internal/notification/SMTPClient"
	"notification/internal/notification/api/handlers"
	wworker "notification/internal/notification/worker"
	rredisClient "notification/internal/redisClient"
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

	config, err := cconfig.New(pathToConfigFile)
	if err != nil {
		log.Fatalf("cannot initialize config: %v", err)
	}

	logger, err := llogger.New(&config.Logger)
	if err != nil {
		log.Fatalf("cannot initialize logger: %v", err)
	}
	defer func() {
		if err = logger.Sync(); err != nil {
			log.Printf("cannot sync logger: %v", err)
		}
	}()

	redisClient, err := rredisClient.New(ctx, &config.Redis, monitoring.New("redis"), logger)
	if err != nil {
		logger.Fatal("cannot initialize redisClient client", zap.Error(err))
	}

	smtpClient := SMTPClient.New(&config.SMTP, logger)

	worker := wworker.New(logger, redisClient, smtpClient, tickTimeForWorker)

	go func() {
		err = worker.Run(ctx)
		if err != nil {
			logger.Error("worker exited with error", zap.Error(err))
		}
	}()

	router := initRouter(logger, &config.Logger, smtpClient, redisClient)

	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%s", config.HttpServer.Host, config.HttpServer.Port),
		Handler: router,
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("metrics available at :2112/metrics")
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("cannot start metrics server: %v", err)
		}
	}()

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
		return
	}

	logger.Info("stopping http server", zap.String("addr", srv.Addr))

	logger.Info("application shutdown completed successfully")
}

func initRouter(logger *zap.Logger, cfg *llogger.Config, smtpClient *SMTPClient.SMTPClient,
	redisClient *rredisClient.RedisCluster) *chi.Mux {
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

// TODO: механизм отказоустойчивости (постоянное переподключение к редису и тд)
// TODO: добавить хранилище отправленных сообщений через PostgreSQL
// TODO: покрыть тестами ВСЁ, нагрузочные тесты?
// TODO: дополнить README, написать документацию
// TODO: добавить в stage сборку прогон тестов

// TODO: GitLab CI/CD
// TODO: попробовать на аккаунте лицея
// TODO: GitHub Actions

// TODO: мониторинг, kubernetes, nginx, kafka
// TODO: многопоточность
// TODO: добавить третий хендлер для множественной отправки единого сообщения на разные адреса?
