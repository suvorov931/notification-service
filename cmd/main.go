package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"notification/internal/config"
	"notification/internal/logger"
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

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(logger.MiddlewareLogger(l))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Post("/", func(w http.ResponseWriter, r *http.Request) {})

	srv := http.Server{
		Addr:    cfg.HttpServer.Addr,
		Handler: router,
	}

	l.Info("starting http server", zap.String("addr:", cfg.HttpServer.Addr))
	if err := srv.ListenAndServe(); err != nil {
		l.Fatal("cannot start http server", zap.Error(err))
	}
}
