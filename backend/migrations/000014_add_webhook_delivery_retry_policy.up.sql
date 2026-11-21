-- Retry policy for webhook deliveries.
--   max_retries      cap on scheduled retries after the first attempt (6 = 1m,5m,15m,1h,6h,24h)
--   retry_count      scheduled retries taken so far; selects the backoff step
--   dead_lettered_at set when the delivery permanently failed (dead-letter queue)
--
-- Existing rows get max_retries 6 and retry_count 0. Rows already in status
-- 'failed' with next_retry_at NULL are treated as terminal and are not retried.
ALTER TABLE webhook_deliveries
    ADD COLUMN IF NOT EXISTS max_retries INTEGER NOT NULL DEFAULT 6,
    ADD COLUMN IF NOT EXISTS retry_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS dead_lettered_at TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_webhook_deliveries_dead_lettered_at ON webhook_deliveries(dead_lettered_at);
