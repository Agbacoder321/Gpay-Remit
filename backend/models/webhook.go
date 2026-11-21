package models

import (
	"time"

	"gorm.io/gorm"
)

type Webhook struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	UserID      uint           `gorm:"index;not null" json:"user_id"`
	URL         string         `gorm:"size:500;not null" json:"url"`
	Secret      string         `gorm:"size:255;not null" json:"-"` // HMAC secret for signature verification
	Events      string         `gorm:"type:text;not null" json:"events"` // Comma-separated list of events
	IsActive    bool           `gorm:"index;default:true" json:"is_active"`
	Description string         `gorm:"size:255" json:"description"`
}

type WebhookDelivery struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	WebhookID     uint           `gorm:"index;not null" json:"webhook_id"`
	Event         string         `gorm:"size:100;not null" json:"event"`
	Payload       string         `gorm:"type:text;not null" json:"payload"`
	Status        string         `gorm:"index;size:20;default:'pending'" json:"status"` // pending, success, failed (failed + next_retry_at = retry scheduled; failed + dead_lettered_at = dead-lettered)
	ResponseCode  int            `json:"response_code"`
	ResponseBody  string         `gorm:"type:text" json:"response_body"`
	ErrorMessage  string         `gorm:"type:text" json:"error_message"`
	AttemptCount  int            `gorm:"default:0" json:"attempt_count"`
	NextRetryAt   *time.Time     `json:"next_retry_at"`
	CompletedAt   *time.Time     `json:"completed_at"`

	// MaxRetries caps scheduled (worker-driven) retries after the initial
	// attempt. Zero means DefaultWebhookMaxRetries.
	MaxRetries int `gorm:"default:6" json:"max_retries"`
	// RetryCount is how many scheduled retries have been scheduled so far.
	// It selects the backoff interval from WebhookRetryBackoff.
	RetryCount int `gorm:"default:0" json:"retry_count"`
	// DeadLetteredAt is set when the delivery has permanently failed (client
	// error, or retries exhausted). Such rows are never retried automatically
	// and form the dead-letter queue.
	DeadLetteredAt *time.Time `gorm:"index" json:"dead_lettered_at"`
}

// DefaultWebhookMaxRetries is the number of scheduled retries after the first
// attempt. It matches the length of WebhookRetryBackoff, so every backoff step
// is used exactly once before the delivery is dead-lettered.
const DefaultWebhookMaxRetries = 6

// WebhookRetryBackoff is the delay before each scheduled retry, in order:
// 1m, 5m, 15m, 1h, 6h, 24h. After the last step the delivery is given up.
var WebhookRetryBackoff = []time.Duration{
	1 * time.Minute,
	5 * time.Minute,
	15 * time.Minute,
	1 * time.Hour,
	6 * time.Hour,
	24 * time.Hour,
}

// TableName overrides the table name
func (Webhook) TableName() string {
	return "webhooks"
}

// TableName overrides the table name
func (WebhookDelivery) TableName() string {
	return "webhook_deliveries"
}
