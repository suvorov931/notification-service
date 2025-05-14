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

	s := service.New(&cfg.CredentialsSender, l)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(logger.MiddlewareLogger(l, &cfg.Logger))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/send-notification", handlers.NewSendNotificationHandler(l, s))

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

// TODO: реализовать функцию для отправки сообщений через время
// TODO: localhost:8080/sending-via-time/...json data...
// TODO: json data: sending time, Mail{}

//curl -X POST http://localhost:8080/ -H "Content-Type: application/json" \
//-d '{
//   "to":"daanisimov04@gmail.com",
//   "subject":"subject",
//   "message":"message"
//}'
