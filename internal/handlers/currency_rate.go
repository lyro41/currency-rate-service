package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (c *CurrencyRateHandler) getCurrencyRate(ctx context.Context, id uuid.UUID, pair string) (db.CurrencyRate, error) {
	if id != uuid.Nil {
		return c.DB.GetCurrencyRateByID(ctx, id)
	}
	return c.DB.GetCurrencyRate(ctx, db.GetCurrencyRateParams{Status: api.StatusFetched, Pair: pair})
}

func (c *CurrencyRateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		parsedID uuid.UUID
		err      error
	)
	id := r.URL.Query().Get("id")
	pair := strings.ToUpper(chi.URLParam(r, "*"))
	resp := api.CurrencyRateResponse{ErrorResponse: api.ErrorResponse{ID: id, Pair: pair}}
	logger := slog.With(slog.String("id", id), slog.String("pair", pair))
	if id == "" {
		if !handlePair(w, r, &resp.ErrorResponse, logger) {
			return
		}
	} else {
		if parsedID, err = uuid.Parse(id); err != nil {
			resp.Error = fmt.Sprintf("invalid id: %s", err)
			writeError(w, r, http.StatusBadRequest, &resp.ErrorResponse)
			logger.Warn(resp.Error)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), c.Timeout)
	defer cancel()
	row, err := c.getCurrencyRate(ctx, parsedID, pair)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			resp.Error = "currency rate has not been fetched yet"
			if id != "" {
				resp.Error = "currency rate request not found"
			}
			writeError(w, r, http.StatusNotFound, &resp.ErrorResponse)
			logger.Info(resp.Error)
			return
		}
		resp.Error = fmt.Sprintf("failed to get currency rate from database: %s", err)
		writeError(w, r, http.StatusInternalServerError, &resp.ErrorResponse)
		logger.Error(resp.Error)
		return
	}
	if id != "" {
		resp.Pair = row.Pair
	}
	render.Status(r, http.StatusOK)
	if row.Rate.Valid {
		resp.Rate = decimal.NewFromBigInt(row.Rate.Int, row.Rate.Exp).String()
	}
	if row.UpdateTime.Valid {
		resp.Time = row.UpdateTime.Time
	}
	resp.Status = row.Status
	render.JSON(w, r, resp)
	logger.Info("successfully queried currency rate", slog.Any("row", row))
	return
}
