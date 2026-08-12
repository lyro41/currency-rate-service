package api

import "github.com/google/uuid"

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
