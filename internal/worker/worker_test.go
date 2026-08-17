package worker

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/lyro41/currency-rate-service/internal/api"
	"github.com/lyro41/currency-rate-service/internal/db"
	"github.com/lyro41/currency-rate-service/internal/provider"
)

type workerTransport struct{ body string }

func (t workerTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(t.body)),
		Header:     make(http.Header),
	}, nil
}

type workerDBTX struct {
	updated chan api.RateStatus
}

func (d workerDBTX) Exec(_ context.Context, _ string, args ...any) (pgconn.CommandTag, error) {
	d.updated <- args[1].(api.RateStatus)
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (workerDBTX) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }

func (workerDBTX) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

func TestDo(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus api.RateStatus
	}{
		{
			name:       "fetched rate",
			body:       `{"date":"2026-08-15","base":"USD","quote":"RUB","rate":92.1234}`,
			wantStatus: api.StatusFetched,
		},
		{name: "provider failure", body: "not-json", wantStatus: api.StatusFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			updated := make(chan api.RateStatus, 1)
			queries := db.New(workerDBTX{updated: updated})
			requests := make(chan api.Request, 1)
			client := &http.Client{Transport: workerTransport{body: tt.body}}

			go Do(ctx, &provider.Provider{Client: client, MaxAttempts: 1}, queries, requests, time.Second)
			requests <- api.Request{UUID: uuid.New(), Pair: "USD/RUB"}
			select {
			case status := <-updated:
				if status != tt.wantStatus {
					t.Fatalf("status = %q, want %q", status, tt.wantStatus)
				}
			case <-time.After(time.Second):
				t.Fatal("worker did not update database")
			}
		})
	}
}
