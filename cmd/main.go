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

	"notification/internal/SMTPClient"
	"notification/internal/api/handlers"
	cconfig "notification/internal/config"
	llogger "notification/internal/logger"
	"notification/internal/monitoring"
	ppostgresClient "notification/internal/storage/postgresClient"
	rredisClient "notification/internal/storage/redisClient"
	wworker "notification/internal/worker"
)

const (
	pathToConfigFile     = "./config/config.env"
	pathToMigrationsFile = "file://./database/migrations"
	tickTimeForWorker    = 1 * time.Second
	shoutdownTime        = 30 * time.Second
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
	defer logger.Sync()

	appMetrics := monitoring.NewAppMetrics()

	redisClient, err := rredisClient.New(ctx, &config.Redis, appMetrics.RedisMetrics, logger)
	if err != nil {
		logger.Fatal("cannot initialize redisClient client", zap.Error(err))
	}

	postgresClient, err := ppostgresClient.New(ctx, &config.Postgres, appMetrics.PostgresMetrics, logger, pathToMigrationsFile)
	if err != nil {
		log.Fatalf("cannot initialize postgres client: %v", err)
	}

	smtpClient := SMTPClient.New(&config.SMTP, appMetrics.SMTPMetrics, logger)

	worker := wworker.New(redisClient, smtpClient, tickTimeForWorker, appMetrics.WorkerMetrics, logger)

	go func() {
		err = worker.Run(ctx)
		if err != nil {
			logger.Error("worker exited with error", zap.Error(err))
		}
	}()

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		monitoringAddr := fmt.Sprintf("%s:%s", config.HttpServer.Host, config.HttpServer.MonitoringPort)
		logger.Info("metrics available at", zap.String("addr", monitoringAddr))
		if err = http.ListenAndServe(monitoringAddr, nil); err != nil {
			logger.Fatal("cannot start metrics server: %v", zap.Error(err))
		}
	}()

	router := initRouter(logger, &config.Logger, smtpClient, redisClient, postgresClient, appMetrics, config.AppTimeouts)

	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%s", config.HttpServer.Host, config.HttpServer.Port),
		Handler: router,
	}

	go func() {
		logger.Info("starting http server", zap.String("addr", srv.Addr))
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("cannot start http server", zap.Error(err))
		}
	}()

	<-ctx.Done()

	gracefulShutdown(logger, &srv, postgresClient, redisClient)
}

func initRouter(logger *zap.Logger, loggerConfig *llogger.Config, smtpClient *SMTPClient.SMTPClient, redisClient *rredisClient.RedisCluster,
	postgresClient *ppostgresClient.PostgresService, appMetrics *monitoring.AppMetrics, timeouts cconfig.AppTimeouts) *chi.Mux {

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(llogger.MiddlewareLogger(logger, loggerConfig))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	notificationHandler := handlers.New(logger, smtpClient, redisClient, postgresClient, timeouts)

	router.Post("/send-notification", notificationHandler.NewSendNotificationHandler(appMetrics.SendNotificationMetrics))

	router.Post("/send-notification-via-time", notificationHandler.NewSendNotificationViaTimeHandler(appMetrics.SendNotificationViaTimeMetrics))

	router.Get("/list", notificationHandler.NewListNotificationHandler(appMetrics.ListNotificationMetrics))

	return router
}

func gracefulShutdown(logger *zap.Logger, srv *http.Server,
	postgresClient ppostgresClient.PostgresClient, redisClient rredisClient.RedisClient) {
	logger.Info("received shutdown signal")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shoutdownTime)
	defer shutdownCancel()

	logger.Info("shutting down http server")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("cannot shutdown http server", zap.Error(err))
		return
	}

	postgresClient.Close()

	err := redisClient.Close()
	if err != nil {
		logger.Error(err.Error())
	}

	logger.Info("stopping http server", zap.String("addr", srv.Addr))

	logger.Info("application shutdown completed successfully")
}

// TODO: механизм отказоустойчивости (постоянное переподключение к редису и тд)
// TODO: дополнить README, написать документацию

// TODO: kubernetes, nginx, kafka
// TODO: многопоточность
// TODO: добавить третий хендлер для множественной отправки единого сообщения на разные адреса?
