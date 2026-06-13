package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/gippuss/devmatch-back/internal/domain"
	"github.com/gippuss/devmatch-back/internal/service"
	"github.com/gippuss/devmatch-back/internal/transport/http/httpx"
	"github.com/gippuss/devmatch-back/internal/transport/http/middleware"
)

type ProjectHandler struct {
	projectService     *service.ProjectService
	applicationService *service.ApplicationService
}

type createProjectRequest struct {
	Title       string  `json:"title" binding:"required,min=3,max=120"`
	Description string  `json:"description" binding:"required,min=10,max=2000"`
	Status      string  `json:"status" binding:"omitempty,oneof=draft recruiting completed banned"`
	TagIDs      []int64 `json:"tag_ids"`
}

type updateProjectRequest struct {
	Title       *string `json:"title" binding:"omitempty,min=3,max=120"`
	Description *string `json:"description" binding:"omitempty,min=10,max=2000"`
	Status      *string `json:"status" binding:"omitempty,oneof=draft recruiting completed banned"`
	TagIDs      []int64 `json:"tag_ids"`
}

type addRoleRequest struct {
	RoleName   string `json:"role_name" binding:"required,min=1,max=100"`
	Grade      string `json:"grade" binding:"omitempty,max=50"`
	SlotsTotal int32  `json:"slots_total" binding:"required,gte=1,lte=100"`
}

type updateRoleRequest struct {
	RoleName   string `json:"role_name" binding:"required,min=1,max=100"`
	Grade      string `json:"grade" binding:"omitempty,max=50"`
	SlotsTotal int32  `json:"slots_total" binding:"required,gte=1,lte=100"`
}

func NewProjectHandler(projectService *service.ProjectService, applicationService *service.ApplicationService) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, applicationService: applicationService}
}

func (h *ProjectHandler) ListProjects(c *gin.Context) {
	filter := domain.ProjectListFilter{
		Query:  strings.TrimSpace(c.Query("q")),
		Status: domain.ProjectStatus(c.Query("status")),
		TagIDs: parseIntSlice(c.Query("tag_ids")),
		Limit:  parseIntDefault(c.Query("limit"), 20),
		Offset: parseIntDefault(c.Query("offset"), 0),
	}

	projects, err := h.projectService.List(c.Request.Context(), filter)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": projects})
}

func (h *ProjectHandler) CreateProject(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	var req createProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	project, err := h.projectService.Create(c.Request.Context(), userID, service.CreateProjectInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      domain.ProjectStatus(req.Status),
		TagIDs:      req.TagIDs,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, project)
}

func (h *ProjectHandler) GetProject(c *gin.Context) {
	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	project, err := h.projectService.GetByID(c.Request.Context(), projectID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, project)
}

func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	var req updateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	var status *domain.ProjectStatus
	if req.Status != nil {
		parsed := domain.ProjectStatus(*req.Status)
		status = &parsed
	}

	project, err := h.projectService.Update(c.Request.Context(), projectID, userID, service.UpdateProjectInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      status,
		TagIDs:      req.TagIDs,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, project)
}

func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	if err := h.projectService.Delete(c.Request.Context(), projectID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ProjectHandler) AddProjectRole(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	var req addRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	role, err := h.projectService.AddRole(c.Request.Context(), projectID, userID, service.AddProjectRoleInput{
		RoleName:   req.RoleName,
		Grade:      req.Grade,
		SlotsTotal: req.SlotsTotal,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusCreated, role)
}

func (h *ProjectHandler) UpdateProjectRole(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	roleID, err := parsePathID(c.Param("role_id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid role id")
		return
	}

	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	role, err := h.projectService.UpdateRole(c.Request.Context(), roleID, userID, service.AddProjectRoleInput{
		RoleName:   req.RoleName,
		Grade:      req.Grade,
		SlotsTotal: req.SlotsTotal,
	})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, role)
}

func (h *ProjectHandler) DeleteProjectRole(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	roleID, err := parsePathID(c.Param("role_id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid role id")
		return
	}

	if err := h.projectService.DeleteRole(c.Request.Context(), roleID, userID); err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ProjectHandler) ListMyProjects(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	projects, err := h.projectService.ListByOwner(c.Request.Context(), userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": projects})
}

func (h *ProjectHandler) ListProjectApplications(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	applications, err := h.applicationService.ListProjectApplications(c.Request.Context(), projectID, userID)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": applications})
}

type submitAppealRequest struct {
	Comment string `json:"comment" binding:"omitempty,max=1000"`
}

func (h *ProjectHandler) SubmitAppeal(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		httpx.WriteError(c, domain.ErrUnauthorized)
		return
	}

	projectID, err := parsePathID(c.Param("id"))
	if err != nil {
		httpx.WriteValidationError(c, "invalid project id")
		return
	}

	var req submitAppealRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.WriteBindError(c, err)
		return
	}

	project, err := h.projectService.SubmitAppeal(c.Request.Context(), projectID, userID, req.Comment)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}

	c.JSON(http.StatusOK, project)
}

func parsePathID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

func parseIntDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}

	return parsed
}

func parseIntSlice(raw string) []int64 {
	if raw == "" {
		return []int64{}
	}

	parts := strings.Split(raw, ",")
	result := make([]int64, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil {
			result = append(result, parsed)
		}
	}

	return result
}
