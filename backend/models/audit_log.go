package models

import "time"

// AuditLog is an immutable, append-only record of a sensitive, state-changing
// operation. The API must never expose update/delete endpoints for this model.
type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`

	UserID    *uint  `gorm:"index" json:"user_id,omitempty"`
	Action    string `gorm:"size:100;not null;index" json:"action"`
	Resource  string `gorm:"size:255;not null;index" json:"resource"`
	OldValue  string `gorm:"type:jsonb" json:"old_value,omitempty"`
	NewValue  string `gorm:"type:jsonb" json:"new_value,omitempty"`
	IPAddress string `gorm:"size:64;not null" json:"ip_address"`
	// ResponseStatus is the HTTP status code the request completed with.
	// Rows written before migration 000013 have 0, meaning "not recorded".
	ResponseStatus int `gorm:"not null;default:0" json:"response_status"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
