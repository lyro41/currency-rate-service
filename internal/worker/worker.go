package worker

import (
	"context"
	"fmt"
	"net/http"
	"os"
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
			status, rate := provider.GetCurrencyRate(client, req.Pair)
			ctx0, cancel := context.WithTimeout(ctx, ctxWaitTime)
			err := q.UpdateCurrencyRate(ctx0, db.UpdateCurrencyRateParams{
				ID:         req.UUID,
				Status:     status,
				Rate:       pgtype.Numeric{Int: rate.Coefficient(), Exp: rate.Exponent(), Valid: true},
				UpdateTime: pgtype.Timestamp{Time: time.Now(), Valid: true},
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "update currency rate: %s\n", err)
			}
			cancel()
		}
	}
}
