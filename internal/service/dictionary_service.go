package service

import (
	"context"

	"github.com/gippuss/devmatch-back/internal/domain"
)

type DictionaryService struct {
	repo DictionaryRepository
}

func NewDictionaryService(repo DictionaryRepository) *DictionaryService {
	return &DictionaryService{repo: repo}
}

func (s *DictionaryService) ListTags(ctx context.Context) ([]domain.Tag, error) {
	return s.repo.ListTags(ctx)
}

func (s *DictionaryService) ListSystemTags(ctx context.Context) ([]domain.Tag, error) {
	return s.repo.ListSystemTags(ctx)
}

func (s *DictionaryService) ListSkills(ctx context.Context) ([]domain.Skill, error) {
	return s.repo.ListSkills(ctx)
}

func (s *DictionaryService) CreateTag(ctx context.Context, name string) (*domain.Tag, error) {
	if len(name) < 1 || len(name) > 64 {
		return nil, domain.ErrValidation
	}
	return s.repo.CreateTag(ctx, name)
}

func (s *DictionaryService) DeleteTagIfUnused(ctx context.Context, tagID int64) error {
	return s.repo.DeleteTagIfUnused(ctx, tagID)
}
