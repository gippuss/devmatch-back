package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gippuss/devmatch-back/internal/domain"
	"github.com/gippuss/devmatch-back/internal/service"
	"github.com/gippuss/devmatch-back/internal/transport/http/httpx"
	"github.com/gippuss/devmatch-back/internal/transport/http/middleware"
)

type MeHandler struct {
	userService        *service.UserService
	applicationService *service.ApplicationService
}

type updateMeRequest struct {
	Username  *string `json:"username" binding:"omitempty,min=2,max=64"`
	Bio       *string `json:"bio" binding:"omitempty,max=500"`
	AvatarURL *string `json:"avatar_url" binding:"omitempty,max=512"`
	SkillIDs  []int64 `json:"skill_ids"`
}

func NewMeHandler(userService *service.UserService, applicationService *service.ApplicationService) *MeHandler {
	return &MeHandler{userService: userService, applicationService: applicationService}
}

func (h *MeHandler) GetMe(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	me, err := h.userService.GetMe(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, me)
}

func (h *MeHandler) GetUserProfile(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		httpx.WriteValidationError(c, "invalid user id")
		return
	}

	user, err := h.userService.GetMe(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *MeHandler) ListMyApplications(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	apps, err := h.applicationService.ListMyApplications(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": apps})
}

func (h *MeHandler) UploadAvatar(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		httpx.WriteValidationError(c, "avatar file is required")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		httpx.WriteValidationError(c, "only jpg, png, webp images are allowed")
		return
	}
	if header.Size > 5*1024*1024 {
		httpx.WriteValidationError(c, "image must be under 5 MB")
		return
	}

	uploadDir := "uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		httpx.WriteError(c, fmt.Errorf("create upload dir: %w", err))
		return
	}

	// Get old avatar to delete it afterwards
	oldUser, _ := h.userService.GetMe(c.Request.Context(), userID)

	filename := fmt.Sprintf("%d_%d%s", userID, time.Now().UnixMilli(), ext)
	dst := filepath.Join(uploadDir, filename)

	out, err := os.Create(dst)
	if err != nil {
		httpx.WriteError(c, fmt.Errorf("create file: %w", err))
		return
	}
	defer out.Close()
	if _, err = io.Copy(out, file); err != nil {
		os.Remove(dst)
		httpx.WriteError(c, fmt.Errorf("write file: %w", err))
		return
	}

	avatarURL := "/uploads/avatars/" + filename
	updated, err := h.userService.UpdateMe(c.Request.Context(), userID, service.UpdateMeInput{
		AvatarURL: &avatarURL,
	})
	if err != nil {
		os.Remove(dst)
		httpx.WriteError(c, err)
		return
	}

	// Delete old local avatar file
	if oldUser != nil && strings.HasPrefix(oldUser.AvatarURL, "/uploads/avatars/") {
		oldPath := strings.TrimPrefix(oldUser.AvatarURL, "/")
		os.Remove(oldPath)
	}

	c.JSON(http.StatusOK, updated)
}

func (h *MeHandler) DeleteAvatar(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	oldUser, _ := h.userService.GetMe(c.Request.Context(), userID)

	empty := ""
	updated, err := h.userService.UpdateMe(c.Request.Context(), userID, service.UpdateMeInput{
		AvatarURL: &empty,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	if oldUser != nil && strings.HasPrefix(oldUser.AvatarURL, "/uploads/avatars/") {
		oldPath := strings.TrimPrefix(oldUser.AvatarURL, "/")
		os.Remove(oldPath)
	}

	c.JSON(http.StatusOK, updated)
}

func (h *MeHandler) UpdateMe(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	me, err := h.userService.UpdateMe(c.Request.Context(), userID, service.UpdateMeInput{
		Username:  req.Username,
		Bio:       req.Bio,
		AvatarURL: req.AvatarURL,
		SkillIDs:  req.SkillIDs,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, me)
}
