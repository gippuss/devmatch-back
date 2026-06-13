package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gippuss/devmatch-back/internal/auth"
)

func TestAuthMiddlewareWithValidAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenManager, err := auth.NewTokenManager(auth.Config{
		AccessSecret:  "access_secret",
		RefreshSecret: "refresh_secret",
		AccessTTL:     time.Minute,
		RefreshTTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("init token manager: %v", err)
	}

	pair, err := tokenManager.IssueTokenPair(42, "jti-1")
	if err != nil {
		t.Fatalf("issue token pair: %v", err)
	}

	router := gin.New()
	router.GET("/private", AuthMiddleware(tokenManager), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddlewareWithInvalidAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenManager, err := auth.NewTokenManager(auth.Config{
		AccessSecret:  "access_secret",
		RefreshSecret: "refresh_secret",
		AccessTTL:     time.Minute,
		RefreshTTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("init token manager: %v", err)
	}

	router := gin.New()
	router.GET("/private", AuthMiddleware(tokenManager), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
