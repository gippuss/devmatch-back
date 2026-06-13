package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gippuss/devmatch-back/internal/domain"
	"github.com/gippuss/devmatch-back/internal/service"
	"github.com/gippuss/devmatch-back/internal/transport/http/httpx"
)

type AdminHandler struct {
	projectService     *service.ProjectService
	applicationService *service.ApplicationService
}

func NewAdminHandler(projectService *service.ProjectService, applicationService *service.ApplicationService) *AdminHandler {
	return &AdminHandler{projectService: projectService, applicationService: applicationService}
}

func (h *AdminHandler) ListAllProjects(c *gin.Context) {
	filter := domain.ProjectListFilter{
		Query:  strings.TrimSpace(c.Query("q")),
		Status: domain.ProjectStatus(c.Query("status")),
		Limit:  parseIntDefault(c.Query("limit"), 100),
		Offset: parseIntDefault(c.Query("offset"), 0),
	}

	projects, err := h.projectService.AdminList(c.Request.Context(), filter)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": projects})
}

type banProjectRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

func (h *AdminHandler) BanProject(c *gin.Context) {
	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	var req banProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	project, err := h.projectService.AdminBan(c.Request.Context(), projectID, req.Reason)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, project)
}

func (h *AdminHandler) UnbanProject(c *gin.Context) {
	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	project, err := h.projectService.AdminUnban(c.Request.Context(), projectID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, project)
}

type reviewAppealRequest struct {
	Approve      bool   `json:"approve"`
	NewBanReason string `json:"new_ban_reason" binding:"omitempty,max=500"`
}

func (h *AdminHandler) ReviewAppeal(c *gin.Context) {
	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	var req reviewAppealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	project, err := h.projectService.ReviewAppeal(c.Request.Context(), projectID, req.Approve, req.NewBanReason)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, project)
}

func (h *AdminHandler) DeleteProject(c *gin.Context) {
	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	if err := h.projectService.AdminDelete(c.Request.Context(), projectID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AdminHandler) ListProjectApplications(c *gin.Context) {
	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	applications, err := h.applicationService.AdminListProjectApplications(c.Request.Context(), projectID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": applications})
}
