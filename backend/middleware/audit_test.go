package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Gpay-Remit/gpay-remit/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.AuditLog{}))
	return db
}

func auditTestRouter(db *gorm.DB, status int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", uint(7))
		c.Next()
	})
	router.Use(AuditTrail(db))
	router.POST("/api/v1/remittances", func(c *gin.Context) {
		SetAuditNew(c, gin.H{"id": 42})
		c.JSON(status, gin.H{"ok": status < 400})
	})
	return router
}

func TestAuditTrail_RecordsResponseStatus(t *testing.T) {
	db := setupAuditTestDB(t)
	router := auditTestRouter(db, http.StatusCreated)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/remittances", strings.NewReader(`{"amount":1}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var logs []models.AuditLog
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, http.StatusCreated, logs[0].ResponseStatus)
	assert.Equal(t, "POST", logs[0].Action)
	assert.Equal(t, "/api/v1/remittances", logs[0].Resource)
	assert.NotEmpty(t, logs[0].NewValue)
}

func TestAuditTrail_RecordsRedirectStatus(t *testing.T) {
	// 3xx is written to the audit log by the middleware (only >=400 is skipped),
	// so the stored code must reflect the real status, not a hard-coded 200.
	db := setupAuditTestDB(t)
	router := auditTestRouter(db, http.StatusSeeOther)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/remittances", strings.NewReader(`{}`))
	router.ServeHTTP(w, req)

	var logs []models.AuditLog
	require.NoError(t, db.Find(&logs).Error)
	require.Len(t, logs, 1)
	assert.Equal(t, http.StatusSeeOther, logs[0].ResponseStatus)
}

func TestAuditTrail_FailedRequestNotWritten(t *testing.T) {
	// Existing behaviour: only successful (2xx/3xx) state changes are audited.
	// Documented here so a change to that policy is a deliberate one.
	db := setupAuditTestDB(t)
	router := auditTestRouter(db, http.StatusBadRequest)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/remittances", strings.NewReader(`{}`))
	router.ServeHTTP(w, req)

	var count int64
	require.NoError(t, db.Model(&models.AuditLog{}).Count(&count).Error)
	assert.Equal(t, int64(0), count)
}
