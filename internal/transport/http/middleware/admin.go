package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/gippuss/devmatch-back/internal/domain"
	"github.com/gippuss/devmatch-back/internal/transport/http/httpx"
)

type UserGetter interface {
	GetMe(ctx context.Context, userID int64) (*domain.User, error)
}

func AdminMiddleware(userGetter UserGetter) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, ok := UserIDFromContext(c)
		if !ok {
			httpx.WriteError(c, domain.ErrUnauthorized)
			return
		}

		user, err := userGetter.GetMe(c.Request.Context(), userID)
		if err != nil {
			httpx.WriteError(c, domain.ErrUnauthorized)
			return
		}

		if !user.IsAdmin {
			httpx.WriteError(c, domain.ErrForbidden)
			return
		}

		c.Next()
	}
}
