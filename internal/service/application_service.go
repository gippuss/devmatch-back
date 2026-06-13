package service

import (
	"context"
	"strings"

	"github.com/gippuss/devmatch-back/internal/domain"
)

type ApplicationService struct {
	repo ApplicationRepository
}

func NewApplicationService(repo ApplicationRepository) *ApplicationService {
	return &ApplicationService{repo: repo}
}

func (s *ApplicationService) Apply(ctx context.Context, userID, projectRoleID int64, message string) (*domain.Application, error) {
	if projectRoleID <= 0 {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "project_role_id must be positive", nil)
	}

	message = strings.TrimSpace(message)
	if message == "" {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "message is required", nil)
	}

	return s.repo.Create(ctx, userID, projectRoleID, message)
}

func (s *ApplicationService) ListProjectApplications(ctx context.Context, projectID, requesterID int64) ([]domain.ApplicationWithUser, error) {
	if projectID <= 0 {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "project id is invalid", nil)
	}

	return s.repo.ListByProjectID(ctx, projectID, requesterID)
}

func (s *ApplicationService) AdminListProjectApplications(ctx context.Context, projectID int64) ([]domain.ApplicationWithUser, error) {
	if projectID <= 0 {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "project id is invalid", nil)
	}
	return s.repo.AdminListByProjectID(ctx, projectID)
}

func (s *ApplicationService) ListMyApplications(ctx context.Context, userID int64) ([]domain.ApplicationWithRole, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *ApplicationService) Review(ctx context.Context, applicationID, ownerID int64, status domain.ApplicationStatus) (*domain.Application, error) {
	if applicationID <= 0 {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "application id is invalid", nil)
	}

	if status != domain.ApplicationStatusAccepted && status != domain.ApplicationStatusRejected {
		return nil, domain.ErrInvalidAppStatus
	}

	return s.repo.UpdateStatus(ctx, applicationID, ownerID, status)
}
