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
	"notification/internal/logger"
	"notification/internal/notification/api/handlers"
	"notification/internal/notification/service"
	"notification/internal/notification/worker"
	"notification/internal/rds"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	defer cancel()

	cfg, err := config.New()
	if err != nil {
		log.Fatalf("cannot initialize config: %v", err)
	}

	l, err := logger.New(&cfg.Logger)
	if err != nil {
		log.Fatalf("cannot initialize logger: %v", err)
	}
	defer l.Sync()

	rc, err := rds.New(ctx, &cfg.Redis, l)
	if err != nil {
		l.Fatal("cannot initialize rds client", zap.Error(err))
	}

	s := service.New(&cfg.MailSender, l)

	w := worker.New(l, rc, s)

	go func() {
		err = w.Run(ctx)
		if err != nil {
			l.Error("failed in the worker's work", zap.Error(err))
		}
	}()

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(logger.MiddlewareLogger(l, &cfg.Logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/send-notification", handlers.NewSendNotificationHandler(l, s))
	router.Post("/send-notification-via-time", handlers.NewSendNotificationViaTimeHandler(l, rc))

	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.HttpServer.Host, cfg.HttpServer.Port),
		Handler: router,
	}

	go func() {
		l.Info("starting http server", zap.String("addr", srv.Addr))
		if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			l.Fatal("cannot start http server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	l.Info("received shutdown signal")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer shutdownCancel()

	l.Info("shutting down http server")
	if err = srv.Shutdown(shutdownCtx); err != nil {
		l.Error("cannot shutdown http server", zap.Error(err))
	} else {
		l.Info("http server shutdown gracefully")
	}

	l.Info("stopping http server", zap.String("addr", srv.Addr))
}

// TODO: везде переехать на testify!!!

// TODO: добавить в stage сборку прогон тестов
// TODO: GitLab CI/CD
// TODO: многопоточность
// TODO: разобраться с отменой на клиентской стороне
// TODO: добавить третий хендлер для множественной отправки единого сообщения на разные адреса?

//curl -X POST http://localhost:8080/send-notification -H "Content-Type: application/json" \
//-d '{
//"to":"daanisimov04@gmail.com",
//"subject":"subject",
//"message":"message"
//}'

//curl -X POST http://localhost:8080/send-notification-via-time -H "Content-Type: application/json" \
//-d '{
//"time":"2035-01-02 15:04:05",
//"to":"daanisimov04@gmail.com",
//"subject":"subject",
//"message":"message"
//}'
