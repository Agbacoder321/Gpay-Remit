# Webhook Event Types

This document describes the webhook events emitted by the Gpay-Remit backend and on-chain Soroban contracts.

## Payload Format

Every webhook delivery sends a JSON POST body:

```json
{
  "event": "payment.completed",
  "timestamp": "2025-01-15T10:30:00Z",
  "data": { ... }
}
```

Each request includes these headers:

| Header | Description |
|--------|-------------|
| `X-Webhook-Signature` | HMAC-SHA256 hex digest of the payload body |
| `X-Webhook-ID` | Numeric webhook ID |
| `Content-Type` | `application/json` |
| `User-Agent` | `GPay-Remit-Webhook/1.0` |

## Event Types

### Payment Events

#### `payment.completed`

Emitted when a remittance is successfully completed.

```json
{
  "event": "payment.completed",
  "timestamp": "2025-01-15T10:30:00Z",
  "data": {
    "remittance_id": "rem_8f3a2b1c",
    "sender_id": "user_42",
    "recipient_id": "user_97",
    "amount": "150.00",
    "currency": "USDC",
    "fee": "2.50",
    "recipient_address": "GCKFJ4Y7S7RQF...",
    "tx_hash": "a1b2c3d4e5f6..."
  }
}
```

#### `payment.failed`

Emitted when a remittance fails or is rejected.

```json
{
  "event": "payment.failed",
  "timestamp": "2025-01-15T10:31:00Z",
  "data": {
    "remittance_id": "rem_8f3a2b1c",
    "sender_id": "user_42",
    "reason": "Insufficient funds in escrow",
    "error_code": "INSUFFICIENT_BALANCE"
  }
}
```

### Escrow Events

#### `escrow.created`

Emitted when a new escrow is opened on-chain.

```json
{
  "event": "escrow.created",
  "timestamp": "2025-01-15T10:00:00Z",
  "data": {
    "escrow_id": "esc_7d9e1f3a",
    "sender": "GCKFJ4Y7S7RQF...",
    "recipient": "GBRT2K3J5...",
    "amount": "500.00",
    "currency": "USDC",
    "deadline": "2025-02-15T23:59:59Z"
  }
}
```

#### `escrow.deposited`

Emitted when funds are deposited into an existing escrow.

```json
{
  "event": "escrow.deposited",
  "timestamp": "2025-01-15T10:05:00Z",
  "data": {
    "escrow_id": "esc_7d9e1f3a",
    "amount": "500.00",
    "depositor": "GCKFJ4Y7S7RQF..."
  }
}
```

#### `escrow.approved`

Emitted when escrow conditions are approved.

```json
{
  "event": "escrow.approved",
  "timestamp": "2025-01-20T14:00:00Z",
  "data": {
    "escrow_id": "esc_7d9e1f3a",
    "approver": "GBRT2K3J5..."
  }
}
```

#### `escrow.released`

Emitted when escrow funds are released to the recipient.

```json
{
  "event": "escrow.released",
  "timestamp": "2025-01-20T14:30:00Z",
  "data": {
    "escrow_id": "esc_7d9e1f3a",
    "recipient": "GBRT2K3J5...",
    "amount": "500.00",
    "tx_hash": "f6e5d4c3b2a1..."
  }
}
```

#### `escrow.refunded`

Emitted when escrow funds are returned to the sender.

```json
{
  "event": "escrow.refunded",
  "timestamp": "2025-02-16T00:00:00Z",
  "data": {
    "escrow_id": "esc_7d9e1f3a",
    "sender": "GCKFJ4Y7S7RQF...",
    "amount": "500.00",
    "reason": "Deadline exceeded"
  }
}
```

#### `escrow.extended`

Emitted when an escrow deadline is extended.

```json
{
  "event": "escrow.extended",
  "timestamp": "2025-02-10T12:00:00Z",
  "data": {
    "escrow_id": "esc_7d9e1f3a",
    "old_deadline": "2025-02-15T23:59:59Z",
    "new_deadline": "2025-03-15T23:59:59Z"
  }
}
```

### Invoice Events

#### `invoice.created`

Emitted when a new invoice is issued.

```json
{
  "event": "invoice.created",
  "timestamp": "2025-01-15T09:00:00Z",
  "data": {
    "invoice_id": "inv_4a5b6c7d",
    "issuer_id": "user_42",
    "amount": "200.00",
    "currency": "USDC",
    "due_date": "2025-02-15",
    "description": "Consulting services - January"
  }
}
```

#### `invoice.paid`

Emitted when an invoice is paid successfully.

```json
{
  "event": "invoice.paid",
  "timestamp": "2025-01-25T16:00:00Z",
  "data": {
    "invoice_id": "inv_4a5b6c7d",
    "payer_id": "user_97",
    "amount": "200.00",
    "currency": "USDC",
    "tx_hash": "b1c2d3e4f5a6..."
  }
}
```

#### `invoice.updated`

Emitted when invoice details are modified.

```json
{
  "event": "invoice.updated",
  "timestamp": "2025-01-18T11:00:00Z",
  "data": {
    "invoice_id": "inv_4a5b6c7d",
    "updated_fields": ["amount", "due_date"],
    "amount": "250.00",
    "due_date": "2025-03-01"
  }
}
```

#### `invoice.cancelled`

Emitted when an invoice is cancelled.

```json
{
  "event": "invoice.cancelled",
  "timestamp": "2025-01-20T08:00:00Z",
  "data": {
    "invoice_id": "inv_4a5b6c7d",
    "reason": "Duplicate invoice",
    "cancelled_by": "user_42"
  }
}
```

#### `invoice.overdue`

Emitted when an invoice passes its due date.

```json
{
  "event": "invoice.overdue",
  "timestamp": "2025-02-16T00:00:00Z",
  "data": {
    "invoice_id": "inv_4a5b6c7d",
    "days_overdue": 1,
    "amount": "200.00"
  }
}
```

### Payment Request Events

#### `payment_request.created`

```json
{
  "event": "payment_request.created",
  "timestamp": "2025-01-15T12:00:00Z",
  "data": {
    "request_id": "pr_9e8d7c6b",
    "sender_id": "user_42",
    "amount": "75.00",
    "currency": "USDC",
    "description": "Service payment"
  }
}
```

#### `payment_request.accepted`

```json
{
  "event": "payment_request.accepted",
  "timestamp": "2025-01-15T13:00:00Z",
  "data": {
    "request_id": "pr_9e8d7c6b",
    "accepted_by": "user_97"
  }
}
```

#### `payment_request.rejected`

```json
{
  "event": "payment_request.rejected",
  "timestamp": "2025-01-15T13:05:00Z",
  "data": {
    "request_id": "pr_9e8d7c6b",
    "rejected_by": "user_97",
    "reason": "Amount mismatch"
  }
}
```

#### `payment_request.cancelled`

```json
{
  "event": "payment_request.cancelled",
  "timestamp": "2025-01-16T10:00:00Z",
  "data": {
    "request_id": "pr_9e8d7c6b",
    "cancelled_by": "user_42"
  }
}
```

### Webhook Delivery Events

#### `webhook.delivery.success`

```json
{
  "event": "webhook.delivery.success",
  "timestamp": "2025-01-15T10:30:01Z",
  "data": {
    "webhook_id": 1,
    "delivery_id": 42,
    "response_code": 200,
    "event_delivered": "payment.completed"
  }
}
```

#### `webhook.delivery.failed`

```json
{
  "event": "webhook.delivery.failed",
  "timestamp": "2025-01-15T11:00:00Z",
  "data": {
    "webhook_id": 1,
    "delivery_id": 43,
    "event_delivered": "payment.completed",
    "error_message": "Connection refused",
    "attempts": 5
  }
}
```

### System Events

#### `admin.action`

```json
{
  "event": "admin.action",
  "timestamp": "2025-01-15T08:00:00Z",
  "data": {
    "admin_id": "user_1",
    "action_type": "rate_limit.reset",
    "target": "192.168.1.100"
  }
}
```

## Wildcard Subscription

Register a webhook with `events: ["*"]` to receive all events.

## Signature Verification

Verify incoming webhooks by computing an HMAC-SHA256 of the raw request body using your webhook secret, then comparing it to the `X-Webhook-Signature` header.

### Node.js

```js
const crypto = require("crypto");

function verifySignature(secret, payload, signature) {
  const expected = crypto.createHmac("sha256", secret).update(payload).digest("hex");
  return crypto.timingSafeEqual(Buffer.from(expected), Buffer.from(signature));
}
```

### Python

```python
import hmac, hashlib

def verify_signature(secret: str, payload: bytes, signature: str) -> bool:
    expected = hmac.new(secret.encode(), payload, hashlib.sha256).hexdigest()
    return hmac.compare_digest(expected, signature)
```

### Go

```go
func VerifySignature(secret, payload, signature string) bool {
    h := hmac.New(sha256.New, []byte(secret))
    h.Write([]byte(payload))
    expected := hex.EncodeToString(h.Sum(nil))
    return hmac.Equal([]byte(expected), []byte(signature))
}
```

## Retry Policy

A delivery gets up to a few quick in-line retries for transient errors. If those are used up, it moves onto a fixed backoff schedule.

**In-line retries (initial delivery only):**

| Error Type | In-line attempts | Base Delay | Backoff |
|------------|------------------|------------|---------|
| Timeout | 3 | 1s | Exponential (1s, 2s, 4s) |
| DNS failure | 2 | 5s | Fixed |
| Connection refused | 0 | — | Goes straight to the scheduled backoff |
| HTTP 5xx | 3 | 1s | Exponential (1s, 2s, 4s) |
| HTTP 4xx | 0 | — | Never retried, dead-lettered immediately |
| Other | 1 | 2s | Goes straight to the scheduled backoff |

**Scheduled backoff (retry worker):** after in-line attempts are exhausted, each retry waits for the next step of this schedule:

| Retry | Delay before retry |
|-------|--------------------|
| 1 | 1 minute |
| 2 | 5 minutes |
| 3 | 15 minutes |
| 4 | 1 hour |
| 5 | 6 hours |
| 6 | 24 hours |

After the sixth scheduled retry fails, the delivery is given up and dead-lettered. Each `webhook_deliveries` row carries `max_retries` (default 6) and `retry_count`, so a delivery can never exceed its budget. Each scheduled attempt makes exactly one request. The worker checks for due deliveries every minute.

**Dead-letter queue:** a delivery that permanently fails (HTTP 4xx, or scheduled retries exhausted) gets `dead_lettered_at` set. It is never retried automatically and stays visible to the owner. A manual retry via `POST /webhooks/deliveries/:delivery_id/retry` resets the budget and takes the delivery out of the queue.

**Metrics:** `gpay_remit_webhook_retry_attempts_total{kind,outcome}` counts each retry attempt (`kind` is `inline` or `scheduled`). `gpay_remit_webhook_dead_letters_total{reason}` counts dead-lettered deliveries (`client_error` or `retries_exhausted`).

## Delivery States

| Status | Meaning |
|--------|---------|
| `pending` | Created, awaiting first delivery attempt |
| `success` | Delivered with a 2xx response |
| `failed` | Last attempt failed. With `next_retry_at` set, a retry is scheduled. With `dead_lettered_at` set, the delivery has permanently failed |
