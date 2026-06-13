package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gippuss/devmatch-back/internal/config"
)

func NewServer(cfg *config.Config, handler *gin.Engine) *http.Server {
	return &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
}
