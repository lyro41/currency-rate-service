package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

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

const maxIdempotencyKeyLength = 255

func NewUpdateHandler(db *db.Queries, queue chan api.Request, timeout time.Duration) *UpdateHandler {
	return &UpdateHandler{
		DB:      db,
		Queue:   queue,
		Timeout: timeout,
	}
}

func (u *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pair := strings.ToUpper(chi.URLParam(r, "*"))
	resp := api.ErrorResponse{Pair: pair}
	logger := slog.With(slog.String("pair", pair))
	if !handlePair(w, r, &resp, logger) {
		return
	}

	id := uuid.New()
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if utf8.RuneCountInString(idempotencyKey) > maxIdempotencyKeyLength {
		resp.Error = "'Idempotency-Key' must be at most 255 characters long"
		writeError(w, r, http.StatusBadRequest, &resp)
		logger.Warn(resp.Error)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), u.Timeout)
	defer cancel()
	var err error
	created := true
	if idempotencyKey == "" {
		_, err = u.DB.CreateUpdateRequest(ctx,
			db.CreateUpdateRequestParams{ID: id, Pair: pair, Status: api.StatusPending})
	} else {
		var row db.CurrencyRate
		row, err = u.DB.CreateIdempotentUpdateRequest(ctx, db.CreateIdempotentUpdateRequestParams{
			ID: id, Pair: pair, Status: api.StatusPending,
			IdempotencyKey: pgtype.Text{String: idempotencyKey, Valid: true},
		})
		if errors.Is(err, pgx.ErrNoRows) {
			row, err = u.DB.GetUpdateRequestByIdempotencyKey(ctx,
				db.GetUpdateRequestByIdempotencyKeyParams{
					Pair: pair, IdempotencyKey: pgtype.Text{String: idempotencyKey, Valid: true}})
			created = false
		}
		if err == nil {
			id = row.ID
		}
	}
	if err != nil {
		resp.Error = "failed to save request to database"
		writeError(w, r, http.StatusInternalServerError, &resp)
		logger.Error("failed to create currency rate request", slog.Any("error", err))
		return
	}
	resp.ID = id.String()
	logger = logger.With("id", id)

	if created {
		select {
		case u.Queue <- api.Request{UUID: id, Pair: pair}:
		default:
			if deleteErr := u.DB.DeletePendingUpdateRequest(ctx, id); deleteErr != nil {
				logger.Error("failed to remove queued request", slog.Any("error", deleteErr))
			}
			resp.ID = ""
			resp.Error = "request queue is full"
			writeError(w, r, http.StatusServiceUnavailable, &resp)
			logger.Warn(resp.Error)
			return
		}
	}
	render.Status(r, http.StatusAccepted)
	render.JSON(w, r, resp)
	logger.Info("successfully created currency rate request")
	return
}
