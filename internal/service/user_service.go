package service

import (
	"context"
	"strings"

	"github.com/gippuss/devmatch-back/internal/domain"
)

type UserService struct {
	repo UserRepository
}

type UpdateMeInput struct {
	Username  *string
	Bio       *string
	AvatarURL *string
	SkillIDs  []int64
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetMe(ctx context.Context, userID int64) (*domain.User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *UserService) UpdateMe(ctx context.Context, userID int64, input UpdateMeInput) (*domain.User, error) {
	if input.Username != nil && strings.TrimSpace(*input.Username) == "" {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "username cannot be blank", nil)
	}

	return s.repo.UpdateProfile(ctx, userID, input.Username, input.Bio, input.AvatarURL, input.SkillIDs)
}
