package api

import (
	"time"

	"github.com/google/uuid"
)

type RateStatus string

func (r RateStatus) String() string { return string(r) }

const (
	StatusPending RateStatus = "pending"
	StatusFetched RateStatus = "fetched"
	StatusFailed  RateStatus = "failed"
)

type Request struct {
	UUID uuid.UUID
	Pair string
}

type ErrorResponse struct {
	Error string `json:"error,omitempty"`
	Pair  string `json:"pair,omitempty"`
	ID    string `json:"id,omitempty"`
}

type CurrencyRateResponse struct {
	ErrorResponse
	Rate   string     `json:"rate,omitempty"`
	Time   time.Time  `json:"time"`
	Status RateStatus `json:"status,omitempty"`
}
