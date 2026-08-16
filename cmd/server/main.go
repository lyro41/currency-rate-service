package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lyro41/currency-rate-service/internal/api"
	"github.com/lyro41/currency-rate-service/internal/config"
	"github.com/lyro41/currency-rate-service/internal/db"
	"github.com/lyro41/currency-rate-service/internal/handlers"
	"github.com/lyro41/currency-rate-service/internal/worker"
)

func main() {
	cfg := config.MustLoad()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %s", err)
	}

	ctx := context.Background()

	slog.Info("initializing database connection")
	pool, err := pgxpool.New(ctx, cfg.Storage.String())
	if err != nil {
		log.Fatalf("connect to database: %s", err)
	}
	defer pool.Close()
	database := db.New(pool)

	slog.Info("initializing database schema")
	_, err = pool.Exec(ctx, db.InitSchema)
	if err != nil {
		log.Fatalf("initialize database schema: %s", err)
	}

	slog.Info("initializing worker")
	client := &http.Client{Timeout: cfg.Provider.Timeout}
	queue := make(chan api.Request, cfg.Worker.BufferSize)
	go worker.Do(ctx, client, database, queue, cfg.Storage.Timeout)

	slog.Info("initializing server")

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(middleware.Timeout(cfg.Provider.Timeout))

	router.Post("/update/*", handlers.NewUpdateHandler(database, queue, cfg.Storage.Timeout).ServeHTTP)
	router.Get("/currency-rate", handlers.NewCurrencyRateHandler(database, cfg.Storage.Timeout).ServeHTTP)
	router.Get("/currency-rate/*", handlers.NewCurrencyRateHandler(database, cfg.Storage.Timeout).ServeHTTP)

	server := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		IdleTimeout:  cfg.IdleTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
	if err = server.ListenAndServe(); err != nil {
		log.Fatalf("listen and serve: %s", err.Error())
	}
}
