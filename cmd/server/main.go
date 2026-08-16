package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		worker.Do(ctx, client, database, queue, cfg.Storage.Timeout)
	}()

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
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err = <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve: %s", err.Error())
		}
	case <-ctx.Done():
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err = server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown: %s", err)
		}
		cancel()

		select {
		case <-workerDone:
		case <-time.After(10 * time.Second):
			log.Print("worker shutdown timed out")
		}
	}
}
