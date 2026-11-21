-- Convert monetary float/decimal columns to bigint minor units (cents/stroops)
ALTER TABLE payments
    ALTER COLUMN amount TYPE BIGINT USING ROUND(amount * 100)::BIGINT,
    ALTER COLUMN converted_amount TYPE BIGINT USING ROUND(COALESCE(converted_amount, 0) * 100)::BIGINT,
    ALTER COLUMN fee TYPE BIGINT USING ROUND(COALESCE(fee, 0) * 100)::BIGINT,
    ALTER COLUMN platform_fee TYPE BIGINT USING ROUND(COALESCE(platform_fee, 0) * 100)::BIGINT,
    ALTER COLUMN forex_fee TYPE BIGINT USING ROUND(COALESCE(forex_fee, 0) * 100)::BIGINT,
    ALTER COLUMN compliance_fee TYPE BIGINT USING ROUND(COALESCE(compliance_fee, 0) * 100)::BIGINT,
    ALTER COLUMN network_fee TYPE BIGINT USING ROUND(COALESCE(network_fee, 0) * 100)::BIGINT;

ALTER TABLE invoices
    ALTER COLUMN amount TYPE BIGINT USING ROUND(amount * 100)::BIGINT;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'payment_requests') THEN
        ALTER TABLE payment_requests
            ALTER COLUMN amount TYPE BIGINT USING ROUND(amount * 100)::BIGINT;
    END IF;
END $$;
