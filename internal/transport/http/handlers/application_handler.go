package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gippuss/devmatch-back/internal/domain"
	"github.com/gippuss/devmatch-back/internal/service"
	"github.com/gippuss/devmatch-back/internal/transport/http/httpx"
	"github.com/gippuss/devmatch-back/internal/transport/http/middleware"
)

type ApplicationHandler struct {
	service *service.ApplicationService
}

type applyRequest struct {
	Message string `json:"message" binding:"required,min=5,max=1000"`
}

type reviewApplicationRequest struct {
	Status string `json:"status" binding:"required,oneof=accepted rejected"`
}

func NewApplicationHandler(service *service.ApplicationService) *ApplicationHandler {
	return &ApplicationHandler{service: service}
}

func (h *ApplicationHandler) Apply(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	projectRoleID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project role id")
		return
	}

	var req applyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	app, err := h.service.Apply(c.Request.Context(), userID, projectRoleID, req.Message)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, app)
}

func (h *ApplicationHandler) Review(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	applicationID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid application id")
		return
	}

	var req reviewApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	app, err := h.service.Review(c.Request.Context(), applicationID, userID, domain.ApplicationStatus(req.Status))
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, app)
}
