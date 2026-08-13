package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5"

	"github.com/lyro41/plata-go-assignment/internal/api"
	"github.com/lyro41/plata-go-assignment/internal/config"
	"github.com/lyro41/plata-go-assignment/internal/db"
	"github.com/lyro41/plata-go-assignment/internal/handlers"
	"github.com/lyro41/plata-go-assignment/internal/worker"
)

func main() {
	cfg := config.MustLoad()
	ctx := context.Background()

	slog.Info("initializing database connection")
	conn, err := pgx.Connect(ctx, cfg.Storage.String())
	if err != nil {
		log.Fatalf("connect to database: %s", err.Error())
	}
	defer conn.Close(ctx)
	database := db.New(conn)

	slog.Info("initializing database schema")
	_, err = conn.Exec(ctx, db.InitSchema)
	if err != nil {
		log.Fatalf("initialize database schema: %s", err.Error())
	}

	slog.Info("initializing worker")
	client := &http.Client{Timeout: cfg.Provider.Timeout}
	queue := make(chan api.Request)
	go worker.Do(ctx, client, database, queue, cfg.Storage.Timeout)

	slog.Info("initializing server")

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)
	router.Use(middleware.Timeout(cfg.Provider.Timeout))

	router.Post("/update/*", handlers.NewUpdateHandler(database, queue, cfg.Storage.Timeout).ServeHTTP)

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
