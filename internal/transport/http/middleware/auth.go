package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gippuss/devmatch-back/internal/auth"
	"github.com/gippuss/devmatch-back/internal/domain"
	"github.com/gippuss/devmatch-back/internal/transport/http/httpx"
)

const contextUserIDKey = "auth_user_id"

func AuthMiddleware(tokenManager *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			httpx.WriteError(c, domain.ErrUnauthorized)
			return
		}

		token := extractBearerToken(authHeader)
		if token == "" {
			httpx.WriteError(c, domain.NewError(http.StatusUnauthorized, domain.ErrUnauthorized.Code, "invalid authorization header", nil))
			return
		}

		claims, err := tokenManager.ParseAccessToken(token)
		if err != nil {
			httpx.WriteError(c, domain.NewError(http.StatusUnauthorized, domain.ErrUnauthorized.Code, "invalid token", err))
			return
		}

		c.Set(contextUserIDKey, claims.UserID)
		c.Next()
	}
}

func UserIDFromContext(c *gin.Context) (int64, bool) {
	value, exists := c.Get(contextUserIDKey)
	if !exists {
		return 0, false
	}

	userID, ok := value.(int64)
	return userID, ok
}

func extractBearerToken(value string) string {
	parts := strings.SplitN(value, " ", 2)
	if len(parts) != 2 {
		return ""
	}

	if strings.ToLower(parts[0]) != "bearer" {
		return ""
	}

	return strings.TrimSpace(parts[1])
}
