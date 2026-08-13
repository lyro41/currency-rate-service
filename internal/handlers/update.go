package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/google/uuid"

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

	Pair string `json:"pair"`
}

func NewUpdateHandler(db *db.Queries, queue chan api.Request, timeout time.Duration) *UpdateHandler {
	return &UpdateHandler{
		DB:      db,
		Queue:   queue,
		Timeout: timeout,
	}
}

var pairRe = regexp.MustCompile(`^([a-zA-Z]+)/([a-zA-Z]+)$`)

func (u *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pair := chi.URLParam(r, "*")
	resp := api.ErrorResponse{Pair: pair}
	logger := slog.With(slog.String("pair", pair))
	if !pairRe.MatchString(pair) {
		resp.Error = "'pair' parameter must be in 'ABC/XYZ' format"
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, resp)
		logger.Warn("invalid pair")
		return
	}

	currencies := pairRe.FindStringSubmatch(pair)
	for _, currency := range currencies[1:] {
		if !validCurrencies[currency] {
			resp.Error = fmt.Sprintf("currency %s is not supported", currency)
			render.Status(r, http.StatusUnprocessableEntity)
			render.JSON(w, r, resp)
			logger.Warn("unsupported currency in pair")
			return
		}
	}

	id := uuid.New()
	resp.ID = id.String()
	logger = logger.With("id", id)
	ctx, cancel := context.WithTimeout(r.Context(), u.Timeout)
	defer cancel()
	_, err := u.DB.CreateUpdateRequest(ctx, db.CreateUpdateRequestParams{ID: id, Pair: pair, Status: api.StatusPending})
	if err != nil {
		resp.Error = "failed to save request to database"
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, resp)
		logger.Error("failed to create currency rate request", slog.Any("error", err))
		return
	}

	u.Queue <- api.Request{UUID: id, Pair: pair}
	render.Status(r, http.StatusOK)
	render.JSON(w, r, resp)
	logger.Info("successfully created currency rate request")
	return
}
