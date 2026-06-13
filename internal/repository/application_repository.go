package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/gippuss/datagate"
	"github.com/gippuss/devmatch-back/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ApplicationRepository struct {
	pool *pgxpool.Pool
	gate datagate.DataGate[applicationRow, applicationFilter]
}

type applicationRow struct {
	ID            int64                    `db:"id"`
	UserID        int64                    `db:"user_id" insert:"user_id"`
	ProjectRoleID int64                    `db:"project_role_id" insert:"project_role_id"`
	Status        domain.ApplicationStatus `db:"status" insert:"status"`
	Message       string                   `db:"message" insert:"message"`
	CreatedAt     time.Time                `db:"created_at" insert:"created_at"`
}

type applicationFilter struct {
	ID *int64 `filter:"id"`
}

func NewApplicationRepository(pool *pgxpool.Pool) (*ApplicationRepository, error) {
	gate, err := datagate.NewDataGate[applicationRow, applicationFilter]("applications", "id", pool)
	if err != nil {
		return nil, err
	}

	return &ApplicationRepository{pool: pool, gate: gate}, nil
}

func (r *ApplicationRepository) Create(ctx context.Context, userID, projectRoleID int64, message string) (*domain.Application, error) {
	appID, err := r.gate.Create(ctx, applicationRow{ID: 0, UserID: userID, ProjectRoleID: projectRoleID, Status: domain.ApplicationStatusPending, Message: message, CreatedAt: time.Now().UTC()})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, domain.NewError(domain.ErrConflict.Status, domain.ErrConflict.Code, "active application already exists", err)
		}
		return nil, fmt.Errorf("create application: %w", err)
	}

	rows, err := r.gate.Get(ctx, applicationFilter{ID: &appID})
	if err != nil {
		return nil, fmt.Errorf("get application after create: %w", err)
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}

	row := rows[0]
	return &domain.Application{ID: row.ID, UserID: row.UserID, ProjectRoleID: row.ProjectRoleID, Status: row.Status, Message: row.Message, CreatedAt: row.CreatedAt}, nil
}

func (r *ApplicationRepository) ListByUserID(ctx context.Context, userID int64) ([]domain.ApplicationWithRole, error) {
	query := `
SELECT a.id, a.user_id, a.project_role_id, a.status, a.message, a.created_at,
       pr.role_name, p.id AS project_id, p.title AS project_title
FROM applications a
JOIN project_roles pr ON pr.id = a.project_role_id
JOIN projects p ON p.id = pr.project_id
WHERE a.user_id = $1
ORDER BY a.created_at DESC
`
	type row struct {
		ID            int64                    `db:"id"`
		UserID        int64                    `db:"user_id"`
		ProjectRoleID int64                    `db:"project_role_id"`
		Status        domain.ApplicationStatus `db:"status"`
		Message       string                   `db:"message"`
		CreatedAt     time.Time                `db:"created_at"`
		RoleName      string                   `db:"role_name"`
		ProjectID     int64                    `db:"project_id"`
		ProjectTitle  string                   `db:"project_title"`
	}

	rows := make([]row, 0)
	if err := pgxscan.Select(ctx, r.pool, &rows, query, userID); err != nil {
		return nil, fmt.Errorf("list applications by user: %w", err)
	}

	result := make([]domain.ApplicationWithRole, 0, len(rows))
	for _, item := range rows {
		result = append(result, domain.ApplicationWithRole{
			Application: domain.Application{
				ID:            item.ID,
				UserID:        item.UserID,
				ProjectRoleID: item.ProjectRoleID,
				Status:        item.Status,
				Message:       item.Message,
				CreatedAt:     item.CreatedAt,
			},
			RoleName:     item.RoleName,
			ProjectID:    item.ProjectID,
			ProjectTitle: item.ProjectTitle,
		})
	}
	return result, nil
}

func (r *ApplicationRepository) ListByProjectID(ctx context.Context, projectID, requesterID int64) ([]domain.ApplicationWithUser, error) {
	ownerQuery := "SELECT owner_id FROM projects WHERE id = $1"
	var ownerID int64
	if err := r.pool.QueryRow(ctx, ownerQuery, projectID).Scan(&ownerID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get project owner: %w", err)
	}
	if ownerID != requesterID {
		return nil, domain.ErrForbidden
	}

	type row struct {
		ID            int64                    `db:"id"`
		UserID        int64                    `db:"user_id"`
		ProjectRoleID int64                    `db:"project_role_id"`
		Status        domain.ApplicationStatus `db:"status"`
		Message       string                   `db:"message"`
		CreatedAt     time.Time                `db:"created_at"`
		Username      string                   `db:"username"`
		AvatarURL     string                   `db:"avatar_url"`
	}

	query := `
SELECT a.id, a.user_id, a.project_role_id, a.status, a.message, a.created_at, u.username, u.avatar_url
FROM applications a
JOIN project_roles pr ON pr.id = a.project_role_id
JOIN users u ON u.id = a.user_id
WHERE pr.project_id = $1
ORDER BY a.created_at DESC
`

	rows := make([]row, 0)
	if err := pgxscan.Select(ctx, r.pool, &rows, query, projectID); err != nil {
		return nil, fmt.Errorf("list project applications: %w", err)
	}

	result := make([]domain.ApplicationWithUser, 0, len(rows))
	for _, item := range rows {
		result = append(result, domain.ApplicationWithUser{
			Application: domain.Application{ID: item.ID, UserID: item.UserID, ProjectRoleID: item.ProjectRoleID, Status: item.Status, Message: item.Message, CreatedAt: item.CreatedAt},
			Applicant:   domain.User{ID: item.UserID, Username: item.Username, AvatarURL: item.AvatarURL},
		})
	}

	return result, nil
}

func (r *ApplicationRepository) AdminListByProjectID(ctx context.Context, projectID int64) ([]domain.ApplicationWithUser, error) {
	type row struct {
		ID            int64                    `db:"id"`
		UserID        int64                    `db:"user_id"`
		ProjectRoleID int64                    `db:"project_role_id"`
		Status        domain.ApplicationStatus `db:"status"`
		Message       string                   `db:"message"`
		CreatedAt     time.Time                `db:"created_at"`
		Username      string                   `db:"username"`
		AvatarURL     string                   `db:"avatar_url"`
	}

	query := `
SELECT a.id, a.user_id, a.project_role_id, a.status, a.message, a.created_at, u.username, u.avatar_url
FROM applications a
JOIN project_roles pr ON pr.id = a.project_role_id
JOIN users u ON u.id = a.user_id
WHERE pr.project_id = $1
ORDER BY a.created_at DESC
`

	rows := make([]row, 0)
	if err := pgxscan.Select(ctx, r.pool, &rows, query, projectID); err != nil {
		return nil, fmt.Errorf("admin list project applications: %w", err)
	}

	result := make([]domain.ApplicationWithUser, 0, len(rows))
	for _, item := range rows {
		result = append(result, domain.ApplicationWithUser{
			Application: domain.Application{ID: item.ID, UserID: item.UserID, ProjectRoleID: item.ProjectRoleID, Status: item.Status, Message: item.Message, CreatedAt: item.CreatedAt},
			Applicant:   domain.User{ID: item.UserID, Username: item.Username, AvatarURL: item.AvatarURL},
		})
	}

	return result, nil
}

func (r *ApplicationRepository) UpdateStatus(ctx context.Context, applicationID, ownerID int64, status domain.ApplicationStatus) (*domain.Application, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin review application tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var app applicationRow
	lockAppQuery := `SELECT id, user_id, project_role_id, status, message, created_at FROM applications WHERE id = $1 FOR UPDATE`
	if err := tx.QueryRow(ctx, lockAppQuery, applicationID).Scan(&app.ID, &app.UserID, &app.ProjectRoleID, &app.Status, &app.Message, &app.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("lock application: %w", err)
	}

	if app.Status != domain.ApplicationStatusPending {
		return nil, domain.ErrInvalidAppStatus
	}

	var projectID, projectOwnerID int64
	var slotsTotal, slotsFilled int32
	lockRoleQuery := `
SELECT pr.project_id, p.owner_id, pr.slots_total, pr.slots_filled
FROM project_roles pr
JOIN projects p ON p.id = pr.project_id
WHERE pr.id = $1
FOR UPDATE
`
	if err := tx.QueryRow(ctx, lockRoleQuery, app.ProjectRoleID).Scan(&projectID, &projectOwnerID, &slotsTotal, &slotsFilled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("lock project role: %w", err)
	}

	if projectOwnerID != ownerID {
		return nil, domain.ErrForbidden
	}

	if status == domain.ApplicationStatusAccepted {
		if slotsFilled >= slotsTotal {
			return nil, domain.ErrNoFreeSlots
		}

		var exists bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2 AND project_role_id = $3)", projectID, app.UserID, app.ProjectRoleID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("check project member exists: %w", err)
		}
		if exists {
			return nil, domain.NewError(domain.ErrConflict.Status, domain.ErrConflict.Code, "user already has this role in project", nil)
		}

		if _, err := tx.Exec(ctx, "UPDATE applications SET status = $1, updated_at = NOW() WHERE id = $2", status, app.ID); err != nil {
			return nil, fmt.Errorf("update application status accepted: %w", err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO project_members (project_id, user_id, project_role_id, joined_at) VALUES ($1, $2, $3, NOW())", projectID, app.UserID, app.ProjectRoleID); err != nil {
			return nil, fmt.Errorf("insert project member: %w", err)
		}
		if _, err := tx.Exec(ctx, "UPDATE project_roles SET slots_filled = slots_filled + 1 WHERE id = $1", app.ProjectRoleID); err != nil {
			return nil, fmt.Errorf("increment slots_filled: %w", err)
		}
	} else {
		if _, err := tx.Exec(ctx, "UPDATE applications SET status = $1, updated_at = NOW() WHERE id = $2", status, app.ID); err != nil {
			return nil, fmt.Errorf("update application status rejected: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit review application tx: %w", err)
	}

	rows, err := r.gate.Get(ctx, applicationFilter{ID: &applicationID})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}

	row := rows[0]
	return &domain.Application{ID: row.ID, UserID: row.UserID, ProjectRoleID: row.ProjectRoleID, Status: row.Status, Message: row.Message, CreatedAt: row.CreatedAt}, nil
}
