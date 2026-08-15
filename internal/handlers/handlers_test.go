package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lyro41/plata-go-assignment/internal/api"
	"github.com/lyro41/plata-go-assignment/internal/db"
)

func TestHandlePair(t *testing.T) {
	tests := []struct {
		name       string
		pair       string
		wantOK     bool
		wantStatus int
		wantError  string
	}{
		{name: "supported pair", pair: "USD/RUB", wantOK: true, wantStatus: http.StatusOK},
		{name: "invalid format", pair: "USDRUB", wantStatus: http.StatusBadRequest, wantError: "'pair' parameter must be in 'ABC/XYZ' format"},
		{name: "unsupported currency", pair: "GBP/RUB", wantStatus: http.StatusUnprocessableEntity, wantError: "currency GBP is not supported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/update/"+tt.pair, nil)
			recorder := httptest.NewRecorder()
			resp := &api.ErrorResponse{Pair: tt.pair}

			ok := handlePair(recorder, req, resp, slog.Default())
			if ok != tt.wantOK {
				t.Fatalf("handlePair() = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantOK {
				if recorder.Code != http.StatusOK {
					t.Fatalf("status = %d, want no error response", recorder.Code)
				}
				return
			}

			if recorder.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			var body api.ErrorResponse
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body.Error != tt.wantError {
				t.Errorf("error = %q, want %q", body.Error, tt.wantError)
			}
		})
	}
}

func TestUpdateHandlerNormalizesPair(t *testing.T) {
	queue := make(chan api.Request, 1)
	handler := NewUpdateHandler(db.New(fakeDBTX{}), queue, time.Second)
	req := httptest.NewRequest(http.MethodPost, "/update/usd/rub", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("*", "usd/rub")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response api.ErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Pair != "USD/RUB" {
		t.Fatalf("response pair = %q, want USD/RUB", response.Pair)
	}
	request := <-queue
	if request.Pair != "USD/RUB" {
		t.Fatalf("queued pair = %q, want USD/RUB", request.Pair)
	}
}

func TestUpdateHandlerIdempotency(t *testing.T) {
	dbtx := &idempotencyDBTX{}
	queue := make(chan api.Request, 2)
	handler := NewUpdateHandler(db.New(dbtx), queue, time.Second)

	request := func() api.ErrorResponse {
		req := httptest.NewRequest(http.MethodPost, "/update/USD/RUB", nil)
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("*", "USD/RUB")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
		req.Header.Set("Idempotency-Key", "same-request")
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		var response api.ErrorResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		return response
	}

	first, second := request(), request()
	if first.ID == "" || first.ID != second.ID {
		t.Fatalf("ids = %q and %q, want the same non-empty id", first.ID, second.ID)
	}
	if len(queue) != 1 {
		t.Fatalf("queued requests = %d, want 1", len(queue))
	}
}

func TestCurrencyRateHandler(t *testing.T) {
	t.Run("by pair", func(t *testing.T) {
		handler := NewCurrencyRateHandler(db.New(currencyRateDBTX{}), time.Second)
		req := httptest.NewRequest(http.MethodGet, "/currency-rate/usd/rub", nil)
		routeCtx := chi.NewRouteContext()
		routeCtx.URLParams.Add("*", "usd/rub")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
		}
		var response api.CurrencyRateResponse
		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.Pair != "USD/RUB" || response.Rate != "92.1234" || response.Status != api.StatusFetched {
			t.Fatalf("unexpected response: %+v", response)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		handler := NewCurrencyRateHandler(db.New(currencyRateDBTX{}), time.Second)
		req := httptest.NewRequest(http.MethodGet, "/currency-rate?id=bad", nil)
		recorder := httptest.NewRecorder()

		handler.ServeHTTP(recorder, req)

		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
		}
	})
}

type fakeDBTX struct{}

func (fakeDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (fakeDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }

func (fakeDBTX) QueryRow(context.Context, string, ...any) pgx.Row { return fakeRow{} }

type fakeRow struct{}

func (fakeRow) Scan(dest ...any) error {
	id := uuid.MustParse("8d8f5e8d-5f1b-4f5e-bf35-3de5c89a1f20")
	*(dest[0].(*uuid.UUID)) = id
	*(dest[1].(*string)) = "USD/RUB"
	return nil
}

type idempotencyDBTX struct{ calls int }

func (d *idempotencyDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (d *idempotencyDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }

func (d *idempotencyDBTX) QueryRow(_ context.Context, query string, _ ...any) pgx.Row {
	if strings.Contains(query, "INSERT INTO currency_rates (id, pair, status, idempotency_key)") {
		d.calls++
		if d.calls == 2 {
			return errorRow{err: pgx.ErrNoRows}
		}
	}
	return fakeRow{}
}

type currencyRateDBTX struct{}

func (currencyRateDBTX) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), nil
}

func (currencyRateDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }

func (currencyRateDBTX) QueryRow(context.Context, string, ...any) pgx.Row { return currencyRateRow{} }

type currencyRateRow struct{}

func (currencyRateRow) Scan(dest ...any) error {
	*(dest[0].(*uuid.UUID)) = uuid.MustParse("8d8f5e8d-5f1b-4f5e-bf35-3de5c89a1f20")
	*(dest[1].(*string)) = "USD/RUB"
	*(dest[2].(*api.RateStatus)) = api.StatusFetched
	*(dest[3].(*pgtype.Numeric)) = pgtype.Numeric{Int: big.NewInt(921234), Exp: -4, Valid: true}
	*(dest[4].(*pgtype.Timestamp)) = pgtype.Timestamp{Valid: true}
	return nil
}

type errorRow struct{ err error }

func (r errorRow) Scan(...any) error { return r.err }
