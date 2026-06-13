package service

import (
	"context"
	"strings"

	"github.com/gippuss/devmatch-back/internal/domain"
)

type ProjectService struct {
	repo ProjectRepository
}

type CreateProjectInput struct {
	Title       string
	Description string
	Status      domain.ProjectStatus
	TagIDs      []int64
}

type UpdateProjectInput struct {
	Title       *string
	Description *string
	Status      *domain.ProjectStatus
	TagIDs      []int64
}

type AddProjectRoleInput struct {
	RoleName   string
	Grade      string
	SlotsTotal int32
}

func NewProjectService(repo ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) Create(ctx context.Context, ownerID int64, input CreateProjectInput) (*domain.Project, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" || input.Description == "" {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "title and description are required", nil)
	}

	if input.Status == "" {
		input.Status = domain.ProjectStatusRecruiting
	}

	return s.repo.Create(ctx, ownerID, input.Title, input.Description, input.Status, input.TagIDs)
}

func (s *ProjectService) List(ctx context.Context, filter domain.ProjectListFilter) ([]domain.Project, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return s.repo.List(ctx, filter)
}

func (s *ProjectService) GetByID(ctx context.Context, projectID int64) (*domain.Project, error) {
	return s.repo.GetByID(ctx, projectID)
}

func (s *ProjectService) ListByOwner(ctx context.Context, ownerID int64) ([]domain.Project, error) {
	return s.repo.ListByOwner(ctx, ownerID)
}

func (s *ProjectService) Update(ctx context.Context, projectID, requesterID int64, input UpdateProjectInput) (*domain.Project, error) {
	if input.Title != nil {
		value := strings.TrimSpace(*input.Title)
		if value == "" {
			return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "title cannot be blank", nil)
		}
		input.Title = &value
	}

	if input.Description != nil {
		value := strings.TrimSpace(*input.Description)
		if value == "" {
			return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "description cannot be blank", nil)
		}
		input.Description = &value
	}

	return s.repo.Update(ctx, projectID, requesterID, input.Title, input.Description, input.Status, input.TagIDs)
}

func (s *ProjectService) Delete(ctx context.Context, projectID, requesterID int64) error {
	return s.repo.Delete(ctx, projectID, requesterID)
}

func (s *ProjectService) AdminList(ctx context.Context, filter domain.ProjectListFilter) ([]domain.Project, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.AdminList(ctx, filter)
}

func (s *ProjectService) AdminDelete(ctx context.Context, projectID int64) error {
	return s.repo.AdminDelete(ctx, projectID)
}

func (s *ProjectService) AdminBan(ctx context.Context, projectID int64, reason string) (*domain.Project, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "ban reason is required", nil)
	}
	return s.repo.AdminBan(ctx, projectID, reason)
}

func (s *ProjectService) AdminUnban(ctx context.Context, projectID int64) (*domain.Project, error) {
	return s.repo.AdminUnban(ctx, projectID)
}

func (s *ProjectService) SubmitAppeal(ctx context.Context, projectID, ownerID int64, comment string) (*domain.Project, error) {
	comment = strings.TrimSpace(comment)
	return s.repo.SubmitAppeal(ctx, projectID, ownerID, comment)
}

func (s *ProjectService) ReviewAppeal(ctx context.Context, projectID int64, approve bool, newBanReason string) (*domain.Project, error) {
	newBanReason = strings.TrimSpace(newBanReason)
	return s.repo.ReviewAppeal(ctx, projectID, approve, newBanReason)
}

func (s *ProjectService) DeleteRole(ctx context.Context, projectRoleID, requesterID int64) error {
	return s.repo.DeleteRole(ctx, projectRoleID, requesterID)
}

func (s *ProjectService) UpdateRole(ctx context.Context, projectRoleID, requesterID int64, input AddProjectRoleInput) (*domain.ProjectRole, error) {
	input.RoleName = strings.TrimSpace(input.RoleName)
	input.Grade = strings.TrimSpace(input.Grade)

	if input.RoleName == "" {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "role_name is required", nil)
	}
	if input.SlotsTotal <= 0 {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "slots_total must be positive", nil)
	}

	return s.repo.UpdateRole(ctx, projectRoleID, requesterID, input.RoleName, input.Grade, input.SlotsTotal)
}

func (s *ProjectService) AddRole(ctx context.Context, projectID, requesterID int64, input AddProjectRoleInput) (*domain.ProjectRole, error) {
	input.RoleName = strings.TrimSpace(input.RoleName)
	input.Grade = strings.TrimSpace(input.Grade)

	if input.RoleName == "" {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "role_name is required", nil)
	}
	if input.SlotsTotal <= 0 {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "slots_total must be positive", nil)
	}

	return s.repo.AddRole(ctx, projectID, requesterID, input.RoleName, input.Grade, input.SlotsTotal)
}
