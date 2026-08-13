CREATE TABLE IF NOT EXISTS currency_rates (
  id UUID PRIMARY KEY,
  pair VARCHAR(10) NOT NULL,
  status VARCHAR(10) NOT NULL,
  rate DECIMAL(20, 8) NULL,
  update_time TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_currency_rates_pair_update_time
ON currency_rates (pair, update_time DESC);
