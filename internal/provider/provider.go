package provider

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/lyro41/plata-go-assignment/internal/api"
	"github.com/shopspring/decimal"
)

type CurrencyRate struct {
	Date  string          `json:"date"`
	Base  string          `json:"base"`
	Quote string          `json:"quote"`
	Rate  decimal.Decimal `json:"rate"`
}

func GetCurrencyRate(client *http.Client, pair string) (api.RateStatus, decimal.Decimal) {
	resp, err := client.Get(fmt.Sprintf("https://api.frankfurter.dev/v2/rate/%s", pair))
	if err != nil {
		fmt.Fprintf(os.Stderr, "get currency rate: %v\n", err)
		return api.StatusFailed, decimal.Decimal{}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse currency rate body: %v\n", err)
		return api.StatusFailed, decimal.Decimal{}
	}
	rate := &CurrencyRate{}
	err = json.Unmarshal(body, rate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal currency rate: %v\n", err)
		return api.StatusFailed, decimal.Decimal{}
	}
	return api.StatusFetched, rate.Rate
}
