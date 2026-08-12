CREATE TYPE RATE_STATUS AS ENUM ('pending', 'fetched', 'failed');

-- name: CreateCurrencyRates
CREATE TABLE IF NOT EXISTS currency_rates (
  id UUID PRIMARY KEY,
  pair VARCHAR(10) NOT NULL,
  status RATE_STATUS NOT NULL,
  rate DECIMAL NULL,
  update_time TIMESTAMP NULL
);

-- name: CreateCurrencyRatesIndex
CREATE INDEX idx_currency_rates_pair_update_time
ON currency_rates (pair, update_time DESC);
