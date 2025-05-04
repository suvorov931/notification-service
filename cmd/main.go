package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"notification/internal/config"
	"notification/internal/logger"
	"notification/internal/notification/api/handler"
	"notification/internal/notification/service"
)

func main() {
	cfg, err := config.New()
	if err != nil {
		log.Fatalf("cannot initialize config: %v", err)
	}

	l, err := logger.New(&cfg.Logger)
	if err != nil {
		log.Fatalf("cannot initialize logger: %v", err)
	}
	defer func() {
		if err := l.Sync(); err != nil {
			log.Fatalf("failed to sync logger: %v", err)
		}
	}()

	s := service.New(cfg, l)

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(logger.MiddlewareLogger(l))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/", handler.NewSendNotificationHandler(l, s))

	srv := http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.HttpServer.Host, cfg.HttpServer.Port),
		Handler: router,
	}

	l.Info("starting http server", zap.String("addr", srv.Addr))
	if err := srv.ListenAndServe(); err != nil {
		l.Fatal("cannot start http server", zap.Error(err))
	}
}

// // TODO: реализовать функцию для отправки сообщений через время
// // TODO: localhost:8080/sending-via-time/...json data...
// // TODO: json data: sending time, notification.Mail{}
//
// // TODO: если на той стороне прервали функцию прервалась и у меня
