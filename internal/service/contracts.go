package service

import (
	"context"
	"time"

	"github.com/gippuss/devmatch-back/internal/domain"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetAuthByEmail(ctx context.Context, email string) (*domain.AuthUser, error)
	GetByID(ctx context.Context, userID int64) (*domain.User, error)
	CreateWithEmail(ctx context.Context, email, passwordHash, username string) (*domain.User, error)
	SetPasswordHash(ctx context.Context, userID int64, passwordHash string) error
	UpdateProfile(ctx context.Context, userID int64, username, bio, avatarURL *string, skillIDs []int64) (*domain.User, error)
}

type ProjectRepository interface {
	Create(ctx context.Context, ownerID int64, title, description string, status domain.ProjectStatus, tagIDs []int64) (*domain.Project, error)
	List(ctx context.Context, filter domain.ProjectListFilter) ([]domain.Project, error)
	ListByOwner(ctx context.Context, ownerID int64) ([]domain.Project, error)
	GetByID(ctx context.Context, projectID int64) (*domain.Project, error)
	Update(ctx context.Context, projectID, requesterID int64, title, description *string, status *domain.ProjectStatus, tagIDs []int64) (*domain.Project, error)
	Delete(ctx context.Context, projectID, requesterID int64) error
	AdminList(ctx context.Context, filter domain.ProjectListFilter) ([]domain.Project, error)
	AdminDelete(ctx context.Context, projectID int64) error
	AdminBan(ctx context.Context, projectID int64, reason string) (*domain.Project, error)
	AdminUnban(ctx context.Context, projectID int64) (*domain.Project, error)
	ReviewAppeal(ctx context.Context, projectID int64, approve bool, newBanReason string) (*domain.Project, error)
	SubmitAppeal(ctx context.Context, projectID, ownerID int64, comment string) (*domain.Project, error)
	AddRole(ctx context.Context, projectID, requesterID int64, roleName, grade string, slotsTotal int32) (*domain.ProjectRole, error)
	UpdateRole(ctx context.Context, projectRoleID, requesterID int64, roleName, grade string, slotsTotal int32) (*domain.ProjectRole, error)
	DeleteRole(ctx context.Context, projectRoleID, requesterID int64) error
}

type ApplicationRepository interface {
	Create(ctx context.Context, userID, projectRoleID int64, message string) (*domain.Application, error)
	ListByProjectID(ctx context.Context, projectID, requesterID int64) ([]domain.ApplicationWithUser, error)
	AdminListByProjectID(ctx context.Context, projectID int64) ([]domain.ApplicationWithUser, error)
	ListByUserID(ctx context.Context, userID int64) ([]domain.ApplicationWithRole, error)
	UpdateStatus(ctx context.Context, applicationID, ownerID int64, status domain.ApplicationStatus) (*domain.Application, error)
}

type DictionaryRepository interface {
	ListTags(ctx context.Context) ([]domain.Tag, error)
	ListSystemTags(ctx context.Context) ([]domain.Tag, error)
	ListSkills(ctx context.Context) ([]domain.Skill, error)
	CreateTag(ctx context.Context, name string) (*domain.Tag, error)
	DeleteTagIfUnused(ctx context.Context, tagID int64) error
}

type RefreshTokenRepository interface {
	Store(ctx context.Context, userID int64, jti string, rawToken string, expiresAt time.Time) error
	ValidateActive(ctx context.Context, jti string, rawToken string) (*domain.StoredRefreshToken, error)
	RevokeByJTI(ctx context.Context, jti string) error
}
