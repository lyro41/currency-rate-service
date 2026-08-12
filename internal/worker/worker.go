package worker

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/lyro41/plata-go-assignment/internal/api"
	"github.com/lyro41/plata-go-assignment/internal/provider"

	"github.com/lyro41/plata-go-assignment/internal/db"
)

var ctxWaitTime = time.Second * 5

func worker(ctx context.Context, client *http.Client, q *db.Queries, requests chan api.Request) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-requests:
			logger := slog.With(slog.String("id", req.UUID.String()), slog.String("pair", req.Pair))

			logger.Info("fetching currency rate")
			status, rate := provider.GetCurrencyRate(client, req.Pair)

			ctx0, cancel := context.WithTimeout(ctx, ctxWaitTime)
			updateTime := time.Now()
			err := q.UpdateCurrencyRate(ctx0, db.UpdateCurrencyRateParams{
				ID:         req.UUID,
				Status:     status,
				Rate:       pgtype.Numeric{Int: rate.Coefficient(), Exp: rate.Exponent(), Valid: true},
				UpdateTime: pgtype.Timestamp{Time: updateTime, Valid: true},
			})
			cancel()

			logger = logger.With(slog.String("status", status.String()), slog.String("rate", rate.String()),
				slog.Time("update_time", updateTime))
			if err != nil {
				logger.Warn("failed to update currency rate", slog.Any("error", err))
				continue
			}
			logger.Info("successfully updated currency rate")
		}
	}
}
