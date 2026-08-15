package provider

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestGetCurrencyRate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.Path != "/v2/rate/USD/RUB" {
				t.Errorf("path = %q, want %q", r.URL.Path, "/v2/rate/USD/RUB")
			}
			return response(`{"date":"2026-08-15","base":"USD","quote":"RUB","rate":92.1234}`), nil
		})}
		status, rate := GetCurrencyRate(client, "USD/RUB")
		if status != "fetched" {
			t.Fatalf("status = %q, want fetched", status)
		}
		if !rate.Equal(decimal.RequireFromString("92.1234")) {
			t.Fatalf("rate = %s, want 92.1234", rate)
		}
	})

	t.Run("transport error", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		})}
		status, rate := GetCurrencyRate(client, "USD/RUB")
		if status != "failed" {
			t.Fatalf("status = %q, want failed", status)
		}
		if !rate.IsZero() {
			t.Fatalf("rate = %s, want zero", rate)
		}
	})

	t.Run("invalid response", func(t *testing.T) {
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return response("not-json"), nil
		})}
		status, rate := GetCurrencyRate(client, "USD/RUB")
		if status != "failed" {
			t.Fatalf("status = %q, want failed", status)
		}
		if !rate.IsZero() {
			t.Fatalf("rate = %s, want zero", rate)
		}
	})
}

func response(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
