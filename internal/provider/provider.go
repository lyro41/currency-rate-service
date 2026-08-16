package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/lyro41/currency-rate-service/internal/api"
	"github.com/shopspring/decimal"
)

type CurrencyRate struct {
	Date  string          `json:"date"`
	Base  string          `json:"base"`
	Quote string          `json:"quote"`
	Rate  decimal.Decimal `json:"rate"`
}

func GetCurrencyRate(client *http.Client, pair string) (api.RateStatus, decimal.Decimal) {
	logger := slog.With(slog.String("pair", pair))

	resp, err := client.Get(fmt.Sprintf("https://api.frankfurter.dev/v2/rate/%s", pair))
	if err != nil {
		logger.Error("get currency rate", slog.Any("error", err))
		return api.StatusFailed, decimal.Decimal{}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Error("get currency rate", slog.Any("resp", resp))
		return api.StatusFailed, decimal.Decimal{}
	}
	logger = logger.With(slog.Any("resp", resp))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("parse currency rate body", slog.Any("error", err))
		return api.StatusFailed, decimal.Decimal{}
	}
	logger = logger.With(slog.String("body", string(body)))

	rate := &CurrencyRate{}
	err = json.Unmarshal(body, rate)
	if err != nil {
		logger.Error("unmarshal currency rate", slog.Any("error", err))
		return api.StatusFailed, decimal.Decimal{}
	}
	if rate.Rate.IsZero() {
		logger.Error("currency rate is zero")
		return api.StatusFailed, decimal.Decimal{}
	}
	return api.StatusFetched, rate.Rate
}
