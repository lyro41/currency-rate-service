package api

import "github.com/google/uuid"

type RateStatus string

const (
	StatusPending RateStatus = "pending"
	StatusFetched RateStatus = "fetched"
	StatusFailed  RateStatus = "failed"
)

type Request struct {
	UUID uuid.UUID
	Pair string
}
