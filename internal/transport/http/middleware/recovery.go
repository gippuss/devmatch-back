package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gippuss/devmatch-back/internal/domain"
	"github.com/gippuss/devmatch-back/internal/transport/http/httpx"
)

func Recovery(log *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Error("panic recovered", zap.Any("error", recovered))
		httpx.WriteError(c, domain.ErrInternal)
	})
}
