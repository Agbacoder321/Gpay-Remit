package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCSRFProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	setupRouter := func() *gin.Engine {
		r := gin.New()
		r.Use(CSRFProtection())
		r.GET("/safe", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		r.POST("/unsafe", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
		return r
	}

	t.Run("safe method sets CSRF cookie", func(t *testing.T) {
		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/safe", nil)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		cookie := w.Header().Get("Set-Cookie")
		assert.Contains(t, cookie, "csrf_token=")
		assert.Contains(t, cookie, "SameSite=Lax")
	})

	t.Run("unsafe method missing cookie returns 403", func(t *testing.T) {
		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/unsafe", nil)
		req.Header.Set("X-CSRF-Token", "some-token")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Missing CSRF cookie")
	})

	t.Run("unsafe method missing header returns 403", func(t *testing.T) {
		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/unsafe", nil)
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-token-123"})
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Missing CSRF header")
	})

	t.Run("unsafe method mismatched token returns 403", func(t *testing.T) {
		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/unsafe", nil)
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "valid-token-123"})
		req.Header.Set("X-CSRF-Token", "wrong-token-456")
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid CSRF token")
	})

	t.Run("unsafe method valid token returns 200", func(t *testing.T) {
		r := setupRouter()
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/unsafe", nil)
		token := "abc123xyz456"
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
		req.Header.Set("X-CSRF-Token", token)
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"status":"ok"`)
	})
}
