package domain

import "time"

type ProjectStatus string

type ApplicationStatus string

const (
	ProjectStatusDraft      ProjectStatus = "draft"
	ProjectStatusRecruiting ProjectStatus = "recruiting"
	ProjectStatusCompleted  ProjectStatus = "completed"
	ProjectStatusBanned     ProjectStatus = "banned"

	ApplicationStatusPending   ApplicationStatus = "pending"
	ApplicationStatusAccepted  ApplicationStatus = "accepted"
	ApplicationStatusRejected  ApplicationStatus = "rejected"
	ApplicationStatusWithdrawn ApplicationStatus = "withdrawn"
)

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Bio       string    `json:"bio"`
	AvatarURL string    `json:"avatar_url"`
	IsAdmin   bool      `json:"is_admin"`
	Skills    []Skill   `json:"skills"`
	CreatedAt time.Time `json:"created_at"`
}

type Skill struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Tag struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	IsSystem bool   `json:"is_system"`
}

type ProjectRole struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	RoleName    string `json:"role_name"`
	Grade       string `json:"grade"`
	SlotsTotal  int32  `json:"slots_total"`
	SlotsFilled int32  `json:"slots_filled"`
}

type AppealStatus string

const (
	AppealStatusNone     AppealStatus = "none"
	AppealStatusPending  AppealStatus = "pending"
	AppealStatusApproved AppealStatus = "approved"
	AppealStatusRejected AppealStatus = "rejected"
)

type Project struct {
	ID                int64         `json:"id"`
	OwnerID           int64         `json:"owner_id"`
	Owner             *User         `json:"owner,omitempty"`
	Title             string        `json:"title"`
	Description       string        `json:"description"`
	Status            ProjectStatus `json:"status"`
	BanReason         string        `json:"ban_reason,omitempty"`
	AppealStatus      AppealStatus  `json:"appeal_status,omitempty"`
	AppealComment     string        `json:"appeal_comment,omitempty"`
	Tags              []Tag         `json:"tags"`
	Roles             []ProjectRole `json:"roles,omitempty"`
	ApplicationsCount int64         `json:"applications_count,omitempty"`
	CreatedAt         time.Time     `json:"created_at"`
}

type Application struct {
	ID            int64             `json:"id"`
	UserID        int64             `json:"user_id"`
	ProjectRoleID int64             `json:"project_role_id"`
	Status        ApplicationStatus `json:"status"`
	Message       string            `json:"message"`
	CreatedAt     time.Time         `json:"created_at"`
}

type ApplicationWithUser struct {
	Application
	Applicant User `json:"applicant"`
}

type ApplicationWithRole struct {
	Application
	RoleName     string `json:"role_name"`
	ProjectID    int64  `json:"project_id"`
	ProjectTitle string `json:"project_title"`
}

type ProjectListFilter struct {
	Query  string
	Status ProjectStatus
	TagIDs []int64
	Limit  int
	Offset int
}

type AuthUser struct {
	ID           int64
	Email        string
	Username     string
	PasswordHash string
}

type StoredRefreshToken struct {
	UserID    int64
	JTI       string
	ExpiresAt time.Time
	RevokedAt *time.Time
}
