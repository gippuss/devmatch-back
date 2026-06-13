package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gippuss/devmatch-back/internal/service"
	"github.com/gippuss/devmatch-back/internal/transport/http/httpx"
	"github.com/gippuss/devmatch-back/internal/transport/http/middleware"
)

type DictionaryHandler struct {
	service *service.DictionaryService
}

func NewDictionaryHandler(service *service.DictionaryService) *DictionaryHandler {
	return &DictionaryHandler{service: service}
}

func (h *DictionaryHandler) ListTags(c *gin.Context) {
	tags, err := h.service.ListTags(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": tags})
}

func (h *DictionaryHandler) ListSystemTags(c *gin.Context) {
	tags, err := h.service.ListSystemTags(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": tags})
}

func (h *DictionaryHandler) ListSkills(c *gin.Context) {
	skills, err := h.service.ListSkills(c.Request.Context())
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": skills})
}

func (h *DictionaryHandler) CreateTag(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteError(c, err)
		return
	}

	tag, err := h.service.CreateTag(c.Request.Context(), req.Name)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, tag)
}

func (h *DictionaryHandler) DeleteTag(c *gin.Context) {
	_, ok := middleware.UserIDFromContext(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}

	tagID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.WriteValidationError(c, "invalid tag id")
		return
	}

	if err := h.service.DeleteTagIfUnused(c.Request.Context(), tagID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
