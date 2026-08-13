package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/lyro41/plata-go-assignment/internal/api"
	"github.com/lyro41/plata-go-assignment/internal/db"
	"github.com/shopspring/decimal"
)

type CurrencyRateHandler struct {
	DB      *db.Queries
	Timeout time.Duration
}

func NewCurrencyRateHandler(db *db.Queries, timeout time.Duration) *CurrencyRateHandler {
	return &CurrencyRateHandler{
		DB:      db,
		Timeout: timeout,
	}
}

func (c *CurrencyRateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		row db.CurrencyRate
		err error
	)
	id := r.URL.Query().Get("id")
	pair := chi.URLParam(r, "*")
	resp := api.CurrencyRateResponse{ErrorResponse: api.ErrorResponse{ID: id, Pair: pair}}
	logger := slog.With(slog.String("id", id), slog.String("pair", pair))
	if id == "" {
		if !handlePair(w, r, &resp.ErrorResponse, logger) {
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), c.Timeout)
		defer cancel()
		row, err = c.DB.GetCurrencyRate(ctx, db.GetCurrencyRateParams{Status: api.StatusFetched, Pair: pair})
		if err != nil {
			resp.Error = fmt.Sprintf("failed to get latest currency rate from database: %s", err)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.ErrorResponse)
			logger.Error(resp.Error)
			return
		}
	} else {
		id, err := uuid.Parse(id)
		if err != nil {
			resp.Error = fmt.Sprintf("invalid id: %s", err)
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ErrorResponse)
			logger.Warn(resp.Error)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), c.Timeout)
		defer cancel()
		row, err = c.DB.GetCurrencyRateByID(ctx, id)
		if err != nil {
			resp.Error = fmt.Sprintf("failed to get currency rate by id from database: %s", err)
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.ErrorResponse)
			logger.Error(resp.Error)
			return
		}
		resp.Pair = row.Pair
	}
	render.Status(r, http.StatusOK)
	resp.Rate = decimal.NewFromBigInt(row.Rate.Int, row.Rate.Exp).String()
	resp.Time = row.UpdateTime.Time
	resp.Status = row.Status
	render.JSON(w, r, resp)
	logger.Info("successfully queried currency rate", slog.Any("row", row))
	return
}
