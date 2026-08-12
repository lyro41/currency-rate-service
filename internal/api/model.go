package api

type RateStatus string

const (
	StatusPending RateStatus = "pending"
	StatusFetched RateStatus = "fetched"
	StatusFailed  RateStatus = "failed"
)
