package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lyro41/plata-go-assignment/internal/api"
	"github.com/lyro41/plata-go-assignment/internal/db"
)

var validCurrencies = map[string]bool{
	"USD": true,
	"EUR": true,
	"MXN": true,
	"RUB": true,
	"JPY": true,
	"AMD": true,
}

type UpdateHandler struct {
	DB      *db.Queries
	Queue   chan api.Request
	Timeout time.Duration
}

func NewUpdateHandler(db *db.Queries, queue chan api.Request, timeout time.Duration) *UpdateHandler {
	return &UpdateHandler{
		DB:      db,
		Queue:   queue,
		Timeout: timeout,
	}
}

func (u *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pair := chi.URLParam(r, "*")
	resp := api.ErrorResponse{Pair: pair}
	logger := slog.With(slog.String("pair", pair))
	if !handlePair(w, r, &resp, logger) {
		return
	}

	id := uuid.New()
	idempotencyKey := r.Header.Get("Idempotency-Key")
	ctx, cancel := context.WithTimeout(r.Context(), u.Timeout)
	defer cancel()
	var err error
	created := true
	if idempotencyKey == "" {
		_, err = u.DB.CreateUpdateRequest(ctx, db.CreateUpdateRequestParams{ID: id, Pair: pair, Status: api.StatusPending})
	} else {
		var row db.CreateUpdateRequestRow
		row, err = u.DB.CreateIdempotentUpdateRequest(ctx, db.CreateIdempotentUpdateRequestParams{
			ID: id, Pair: pair, Status: api.StatusPending, IdempotencyKey: idempotencyKey,
		})
		if err == pgx.ErrNoRows {
			row, err = u.DB.GetUpdateRequestByIdempotencyKey(ctx, pair, idempotencyKey)
			created = false
		}
		if err == nil {
			id = row.ID
		}
	}
	if err != nil {
		resp.Error = "failed to save request to database"
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, resp)
		logger.Error("failed to create currency rate request", slog.Any("error", err))
		return
	}
	resp.ID = id.String()
	logger = logger.With("id", id)

	if created {
		u.Queue <- api.Request{UUID: id, Pair: pair}
	}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, resp)
	logger.Info("successfully created currency rate request")
	return
}
