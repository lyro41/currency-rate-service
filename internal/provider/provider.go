package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lyro41/currency-rate-service/internal/api"
)

type Provider struct {
	Client         *http.Client
	MaxAttempts    int
	InitialBackoff time.Duration
}

type CurrencyRate struct {
	Date  string          `json:"date"`
	Base  string          `json:"base"`
	Quote string          `json:"quote"`
	Rate  decimal.Decimal `json:"rate"`
}

func (p *Provider) GetCurrencyRate(ctx context.Context, pair string) (api.RateStatus, decimal.Decimal) {
	logger := slog.With(slog.String("pair", pair))

	for attempt := range p.MaxAttempts {
		status, rate, retry := p.getCurrencyRate(ctx, pair, logger, attempt)
		if !retry || attempt == p.MaxAttempts-1 {
			return status, rate
		}
		backoff := p.InitialBackoff * time.Duration(1<<attempt)
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return api.StatusFailed, decimal.Decimal{}
		case <-timer.C:
		}
	}
	return api.StatusFailed, decimal.Decimal{}
}

func (p *Provider) getCurrencyRate(
	ctx context.Context, pair string, logger *slog.Logger, attempt int) (api.RateStatus, decimal.Decimal, bool) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("https://api.frankfurter.dev/v2/rate/%s", pair), nil)
	if err != nil {
		return api.StatusFailed, decimal.Decimal{}, false
	}
	resp, err := p.Client.Do(req)
	if err != nil {
		logger.Warn("get currency rate failed, retrying", slog.Int("attempt", attempt), slog.Any("error", err))
		return api.StatusFailed, decimal.Decimal{}, true
	}
	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		logger.Warn("read currency rate response failed, retrying", slog.Int("attempt", attempt),
			slog.Any("error", readErr))
		return api.StatusFailed, decimal.Decimal{}, true
	}
	if resp.StatusCode != http.StatusOK {
		retry := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError
		if retry {
			logger.Warn("currency rate provider returned retryable status",
				slog.Int("attempt", attempt), slog.Int("status", resp.StatusCode))
		}
		return api.StatusFailed, decimal.Decimal{}, retry
	}

	rate := &CurrencyRate{}
	if err := json.Unmarshal(body, rate); err != nil {
		logger.Error("unmarshal currency rate", slog.Any("error", err))
		return api.StatusFailed, decimal.Decimal{}, false
	}
	if rate.Rate.IsZero() {
		logger.Error("currency rate is zero")
		return api.StatusFailed, decimal.Decimal{}, false
	}
	return api.StatusFetched, rate.Rate, false
}
