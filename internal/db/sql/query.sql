-- name: GetCurrencyRate :one
SELECT * FROM currency_rates
WHERE status = $1 AND pair = $2
ORDER BY update_time DESC
LIMIT 1;

-- name: GetCurrencyRateByID :one
SELECT * FROM currency_rates
WHERE id = $1;

-- name: CreateUpdateRequest :one
INSERT INTO currency_rates (id, pair, status)
VALUES ($1, $2, $3)
RETURNING id, pair;

-- name: UpdateCurrencyRate :exec
UPDATE currency_rates
SET status = $2, rate = $3, update_time = $4
WHERE id = $1;
