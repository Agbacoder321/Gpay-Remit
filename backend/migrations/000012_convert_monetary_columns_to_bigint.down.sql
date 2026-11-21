ALTER TABLE payments
    ALTER COLUMN amount TYPE DECIMAL USING (amount::DECIMAL / 100.0),
    ALTER COLUMN converted_amount TYPE DECIMAL USING (converted_amount::DECIMAL / 100.0),
    ALTER COLUMN fee TYPE DECIMAL USING (fee::DECIMAL / 100.0),
    ALTER COLUMN platform_fee TYPE DECIMAL USING (platform_fee::DECIMAL / 100.0),
    ALTER COLUMN forex_fee TYPE DECIMAL USING (forex_fee::DECIMAL / 100.0),
    ALTER COLUMN compliance_fee TYPE DECIMAL USING (compliance_fee::DECIMAL / 100.0),
    ALTER COLUMN network_fee TYPE DECIMAL USING (network_fee::DECIMAL / 100.0);

ALTER TABLE invoices
    ALTER COLUMN amount TYPE DECIMAL USING (amount::DECIMAL / 100.0);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'payment_requests') THEN
        ALTER TABLE payment_requests
            ALTER COLUMN amount TYPE DECIMAL USING (amount::DECIMAL / 100.0);
    END IF;
END $$;
