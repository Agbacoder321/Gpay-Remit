DROP INDEX IF EXISTS idx_webhook_deliveries_dead_lettered_at;

ALTER TABLE webhook_deliveries
    DROP COLUMN IF EXISTS dead_lettered_at,
    DROP COLUMN IF EXISTS retry_count,
    DROP COLUMN IF EXISTS max_retries;
