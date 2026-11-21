-- Records the HTTP status code of the audited request so an auditor can
-- tell succeeded from failed operations. 0 = not recorded (pre-migration rows).
ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS response_status INTEGER NOT NULL DEFAULT 0;
