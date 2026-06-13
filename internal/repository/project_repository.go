package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gippuss/datagate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gippuss/devmatch-back/internal/domain"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type ProjectRepository struct {
	pool             *pgxpool.Pool
	projectsGate     datagate.DataGate[projectRow, projectFilter]
	projectRolesGate datagate.DataGate[projectRoleRow, projectRoleFilter]
}

type projectRow struct {
	ID            int64                `db:"id"`
	OwnerID       int64                `db:"owner_id" insert:"owner_id"`
	Title         string               `db:"title" insert:"title"`
	Description   string               `db:"description" insert:"description"`
	Status        domain.ProjectStatus `db:"status" insert:"status"`
	BanReason     string               `db:"ban_reason"`
	AppealStatus  domain.AppealStatus  `db:"appeal_status"`
	AppealComment string               `db:"appeal_comment"`
	CreatedAt     time.Time            `db:"created_at" insert:"created_at"`
}

type projectFilter struct {
	ID *int64 `filter:"id"`
}

type projectRoleRow struct {
	ID          int64     `db:"id"`
	ProjectID   int64     `db:"project_id" insert:"project_id"`
	RoleName    string    `db:"role_name" insert:"role_name"`
	Grade       string    `db:"grade" insert:"grade"`
	SlotsTotal  int32     `db:"slots_total" insert:"slots_total"`
	SlotsFilled int32     `db:"slots_filled" insert:"slots_filled"`
	CreatedAt   time.Time `db:"created_at" insert:"created_at"`
}

type projectRoleFilter struct {
	ID *int64 `filter:"id"`
}

func NewProjectRepository(pool *pgxpool.Pool) (*ProjectRepository, error) {
	projectsGate, err := datagate.NewDataGate[projectRow, projectFilter]("projects", "id", pool)
	if err != nil {
		return nil, err
	}

	projectRolesGate, err := datagate.NewDataGate[projectRoleRow, projectRoleFilter]("project_roles", "id", pool)
	if err != nil {
		return nil, err
	}

	return &ProjectRepository{pool: pool, projectsGate: projectsGate, projectRolesGate: projectRolesGate}, nil
}

func (r *ProjectRepository) Create(ctx context.Context, ownerID int64, title, description string, status domain.ProjectStatus, tagIDs []int64) (*domain.Project, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx for create project: %w", err)
	}
	defer tx.Rollback(ctx)

	projectID, err := r.projectsGate.GetWithTransaction(tx).Create(ctx, projectRow{ID: 0, OwnerID: ownerID, Title: title, Description: description, Status: status, CreatedAt: time.Now().UTC()})
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	if len(tagIDs) > 0 {
		q := psql.Insert("project_tags").Columns("project_id", "tag_id")
		for _, tagID := range tagIDs {
			q = q.Values(projectID, tagID)
		}
		sql, args, err := q.ToSql()
		if err != nil {
			return nil, fmt.Errorf("build insert project tags: %w", err)
		}
		if _, err := tx.Exec(ctx, sql, args...); err != nil {
			return nil, fmt.Errorf("insert project tags: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create project: %w", err)
	}

	return r.GetByID(ctx, projectID)
}

func (r *ProjectRepository) List(ctx context.Context, filter domain.ProjectListFilter) ([]domain.Project, error) {
	builder := psql.Select(
		"p.id", "p.owner_id", "p.title", "p.description", "p.status", "p.ban_reason", "p.appeal_status", "p.appeal_comment", "p.created_at",
		"u.id", "u.username", "u.email", "u.bio", "u.avatar_url", "u.created_at",
	).
		From("projects p").
		Join("users u ON u.id = p.owner_id").
		OrderBy("p.created_at DESC").
		Limit(uint64(filter.Limit)).
		Offset(uint64(filter.Offset))

	if filter.Status != "" {
		builder = builder.Where(sq.Eq{"p.status": filter.Status})
	} else {
		builder = builder.Where(sq.NotEq{"p.status": domain.ProjectStatusBanned})
	}
	if filter.Query != "" {
		like := "%" + strings.ToLower(filter.Query) + "%"
		builder = builder.Where(sq.Or{sq.Expr("LOWER(p.title) LIKE ?", like), sq.Expr("LOWER(p.description) LIKE ?", like)})
	}
	if len(filter.TagIDs) > 0 {
		builder = builder.Where(sq.Expr("EXISTS (SELECT 1 FROM project_tags pt WHERE pt.project_id = p.id AND pt.tag_id = ANY(?))", filter.TagIDs))
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build project list query: %w", err)
	}

	listRows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer listRows.Close()

	result := make([]domain.Project, 0)
	for listRows.Next() {
		var p domain.Project
		var u domain.User
		if err := listRows.Scan(
			&p.ID, &p.OwnerID, &p.Title, &p.Description, &p.Status, &p.BanReason, &p.AppealStatus, &p.AppealComment, &p.CreatedAt,
			&u.ID, &u.Username, &u.Email, &u.Bio, &u.AvatarURL, &u.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project list row: %w", err)
		}
		p.Owner = &u
		result = append(result, p)
	}
	if err := listRows.Err(); err != nil {
		return nil, err
	}

	for i := range result {
		tags, err := r.listTagsByProjectID(ctx, result[i].ID)
		if err != nil {
			return nil, err
		}
		result[i].Tags = tags
	}

	return result, nil
}

func (r *ProjectRepository) GetByID(ctx context.Context, projectID int64) (*domain.Project, error) {
	rows, err := r.projectsGate.Get(ctx, projectFilter{ID: &projectID})
	if err != nil {
		return nil, fmt.Errorf("get project by id: %w", err)
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}

	row := rows[0]
	project := &domain.Project{ID: row.ID, OwnerID: row.OwnerID, Title: row.Title, Description: row.Description, Status: row.Status, BanReason: row.BanReason, AppealStatus: row.AppealStatus, AppealComment: row.AppealComment, CreatedAt: row.CreatedAt}

	sql, args, err := psql.Select("id", "email", "username", "bio", "avatar_url", "created_at").
		From("users").
		Where(sq.Eq{"id": row.OwnerID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get owner query: %w", err)
	}
	var owner domain.User
	if err := r.pool.QueryRow(ctx, sql, args...).Scan(&owner.ID, &owner.Email, &owner.Username, &owner.Bio, &owner.AvatarURL, &owner.CreatedAt); err == nil {
		project.Owner = &owner
	}

	tags, err := r.listTagsByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	project.Tags = tags

	roles, err := r.listProjectRoles(ctx, projectID)
	if err != nil {
		return nil, err
	}
	project.Roles = roles

	return project, nil
}

func (r *ProjectRepository) Update(ctx context.Context, projectID, requesterID int64, title, description *string, status *domain.ProjectStatus, tagIDs []int64) (*domain.Project, error) {
	project, err := r.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerID != requesterID {
		return nil, domain.ErrForbidden
	}

	updateData := map[string]any{}
	if title != nil {
		updateData["title"] = *title
	}
	if description != nil {
		updateData["description"] = *description
	}
	if status != nil {
		if project.Status == domain.ProjectStatusBanned {
			return nil, domain.NewError(400, "invalid_state", "cannot change status of a banned project", nil)
		}
		updateData["status"] = *status
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update project tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if len(updateData) > 0 {
		if err := r.projectsGate.GetWithTransaction(tx).Update(ctx, projectFilter{ID: &projectID}, updateData); err != nil && !errors.Is(err, datagate.ErrNoRowsAffected) {
			return nil, fmt.Errorf("update project data: %w", err)
		}
	}

	var removedTagIDs []int64
	if tagIDs != nil {
		selectSQL, selectArgs, err := psql.Select("tag_id").
			From("project_tags").
			Where(sq.Eq{"project_id": projectID}).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("build get old project tags query: %w", err)
		}
		oldRows, err := r.pool.Query(ctx, selectSQL, selectArgs...)
		if err != nil {
			return nil, fmt.Errorf("get old project tags: %w", err)
		}
		newTagSet := make(map[int64]struct{}, len(tagIDs))
		for _, id := range tagIDs {
			newTagSet[id] = struct{}{}
		}
		for oldRows.Next() {
			var tid int64
			if err := oldRows.Scan(&tid); err != nil {
				oldRows.Close()
				return nil, fmt.Errorf("scan old tag id: %w", err)
			}
			if _, kept := newTagSet[tid]; !kept {
				removedTagIDs = append(removedTagIDs, tid)
			}
		}
		oldRows.Close()

		deleteSQL, deleteArgs, err := psql.Delete("project_tags").
			Where(sq.Eq{"project_id": projectID}).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("build clear project tags query: %w", err)
		}
		if _, err := tx.Exec(ctx, deleteSQL, deleteArgs...); err != nil {
			return nil, fmt.Errorf("clear project tags: %w", err)
		}

		if len(tagIDs) > 0 {
			insertQ := psql.Insert("project_tags").Columns("project_id", "tag_id")
			for _, tagID := range tagIDs {
				insertQ = insertQ.Values(projectID, tagID)
			}
			insertSQL, insertArgs, err := insertQ.ToSql()
			if err != nil {
				return nil, fmt.Errorf("build insert project tags query: %w", err)
			}
			if _, err := tx.Exec(ctx, insertSQL, insertArgs...); err != nil {
				return nil, fmt.Errorf("insert project tags: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update project: %w", err)
	}

	for _, tid := range removedTagIDs {
		cleanSQL, cleanArgs, err := psql.Delete("tags").
			Where(sq.Eq{"id": tid}).
			Where(sq.Eq{"is_system": false}).
			Where(sq.Expr("NOT EXISTS (SELECT 1 FROM project_tags WHERE tag_id = ?)", tid)).
			ToSql()
		if err == nil {
			_, _ = r.pool.Exec(ctx, cleanSQL, cleanArgs...)
		}
	}

	return r.GetByID(ctx, projectID)
}

func (r *ProjectRepository) Delete(ctx context.Context, projectID, requesterID int64) error {
	project, err := r.GetByID(ctx, projectID)
	if err != nil {
		return err
	}
	if project.OwnerID != requesterID {
		return domain.ErrForbidden
	}

	selectSQL, selectArgs, err := psql.Select("pt.tag_id").
		From("project_tags pt").
		Join("tags t ON t.id = pt.tag_id").
		Where(sq.Eq{"pt.project_id": projectID}).
		Where(sq.Eq{"t.is_system": false}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build get custom tags query: %w", err)
	}

	var customTagIDs []int64
	rows, err := r.pool.Query(ctx, selectSQL, selectArgs...)
	if err == nil {
		for rows.Next() {
			var tid int64
			if rows.Scan(&tid) == nil {
				customTagIDs = append(customTagIDs, tid)
			}
		}
		rows.Close()
	}

	if err := r.projectsGate.Delete(ctx, projectFilter{ID: &projectID}); err != nil {
		if errors.Is(err, datagate.ErrNoRowsAffected) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete project: %w", err)
	}

	for _, tid := range customTagIDs {
		cleanSQL, cleanArgs, err := psql.Delete("tags").
			Where(sq.Eq{"id": tid}).
			Where(sq.Eq{"is_system": false}).
			Where(sq.Expr("NOT EXISTS (SELECT 1 FROM project_tags WHERE tag_id = ?)", tid)).
			ToSql()
		if err == nil {
			_, _ = r.pool.Exec(ctx, cleanSQL, cleanArgs...)
		}
	}

	return nil
}

func (r *ProjectRepository) AdminList(ctx context.Context, filter domain.ProjectListFilter) ([]domain.Project, error) {
	builder := psql.Select(
		"p.id", "p.owner_id", "p.title", "p.description", "p.status", "p.ban_reason", "p.appeal_status", "p.appeal_comment", "p.created_at",
		"u.id", "u.username", "u.email", "u.bio", "u.avatar_url", "u.created_at",
		"(SELECT COUNT(*) FROM applications a JOIN project_roles pr ON pr.id = a.project_role_id WHERE pr.project_id = p.id) AS applications_count",
	).
		From("projects p").
		Join("users u ON u.id = p.owner_id").
		OrderBy("p.created_at DESC").
		Limit(uint64(filter.Limit)).
		Offset(uint64(filter.Offset))

	if filter.Status != "" {
		builder = builder.Where(sq.Eq{"p.status": filter.Status})
	}
	if filter.Query != "" {
		like := "%" + strings.ToLower(filter.Query) + "%"
		builder = builder.Where(sq.Or{sq.Expr("LOWER(p.title) LIKE ?", like), sq.Expr("LOWER(p.description) LIKE ?", like)})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build admin project list query: %w", err)
	}

	listRows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("admin list projects: %w", err)
	}
	defer listRows.Close()

	result := make([]domain.Project, 0)
	for listRows.Next() {
		var p domain.Project
		var u domain.User
		if err := listRows.Scan(
			&p.ID, &p.OwnerID, &p.Title, &p.Description, &p.Status, &p.BanReason, &p.AppealStatus, &p.AppealComment, &p.CreatedAt,
			&u.ID, &u.Username, &u.Email, &u.Bio, &u.AvatarURL, &u.CreatedAt,
			&p.ApplicationsCount,
		); err != nil {
			return nil, fmt.Errorf("scan admin project list row: %w", err)
		}
		p.Owner = &u
		result = append(result, p)
	}
	if err := listRows.Err(); err != nil {
		return nil, err
	}

	for i := range result {
		tags, err := r.listTagsByProjectID(ctx, result[i].ID)
		if err != nil {
			return nil, err
		}
		result[i].Tags = tags
	}

	return result, nil
}

func (r *ProjectRepository) AdminUpdate(ctx context.Context, projectID int64, title, description *string, status *domain.ProjectStatus, tagIDs []int64) (*domain.Project, error) {
	updateData := map[string]any{}
	if title != nil {
		updateData["title"] = *title
	}
	if description != nil {
		updateData["description"] = *description
	}
	if status != nil {
		updateData["status"] = *status
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin admin update project tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if len(updateData) > 0 {
		if err := r.projectsGate.GetWithTransaction(tx).Update(ctx, projectFilter{ID: &projectID}, updateData); err != nil && !errors.Is(err, datagate.ErrNoRowsAffected) {
			return nil, fmt.Errorf("admin update project data: %w", err)
		}
	}

	var removedTagIDs []int64
	if tagIDs != nil {
		selectSQL, selectArgs, err := psql.Select("tag_id").
			From("project_tags").
			Where(sq.Eq{"project_id": projectID}).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("build get old project tags query: %w", err)
		}
		oldRows, err := r.pool.Query(ctx, selectSQL, selectArgs...)
		if err != nil {
			return nil, fmt.Errorf("get old project tags: %w", err)
		}
		newTagSet := make(map[int64]struct{}, len(tagIDs))
		for _, id := range tagIDs {
			newTagSet[id] = struct{}{}
		}
		for oldRows.Next() {
			var tid int64
			if err := oldRows.Scan(&tid); err != nil {
				oldRows.Close()
				return nil, fmt.Errorf("scan old tag id: %w", err)
			}
			if _, kept := newTagSet[tid]; !kept {
				removedTagIDs = append(removedTagIDs, tid)
			}
		}
		oldRows.Close()

		deleteSQL, deleteArgs, err := psql.Delete("project_tags").
			Where(sq.Eq{"project_id": projectID}).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("build clear project tags query: %w", err)
		}
		if _, err := tx.Exec(ctx, deleteSQL, deleteArgs...); err != nil {
			return nil, fmt.Errorf("clear project tags: %w", err)
		}

		if len(tagIDs) > 0 {
			insertQ := psql.Insert("project_tags").Columns("project_id", "tag_id")
			for _, tagID := range tagIDs {
				insertQ = insertQ.Values(projectID, tagID)
			}
			insertSQL, insertArgs, err := insertQ.ToSql()
			if err != nil {
				return nil, fmt.Errorf("build insert project tags query: %w", err)
			}
			if _, err := tx.Exec(ctx, insertSQL, insertArgs...); err != nil {
				return nil, fmt.Errorf("insert project tags: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit admin update project: %w", err)
	}

	for _, tid := range removedTagIDs {
		cleanSQL, cleanArgs, err := psql.Delete("tags").
			Where(sq.Eq{"id": tid}).
			Where(sq.Eq{"is_system": false}).
			Where(sq.Expr("NOT EXISTS (SELECT 1 FROM project_tags WHERE tag_id = ?)", tid)).
			ToSql()
		if err == nil {
			_, _ = r.pool.Exec(ctx, cleanSQL, cleanArgs...)
		}
	}

	return r.GetByID(ctx, projectID)
}

func (r *ProjectRepository) AdminAddRole(ctx context.Context, projectID int64, roleName, grade string, slotsTotal int32) (*domain.ProjectRole, error) {
	projectRoleID, err := r.projectRolesGate.Create(ctx, projectRoleRow{
		ProjectID:   projectID,
		RoleName:    roleName,
		Grade:       grade,
		SlotsTotal:  slotsTotal,
		SlotsFilled: 0,
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("admin add role: %w", err)
	}
	roles, err := r.listProjectRoles(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.ID == projectRoleID {
			return &role, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *ProjectRepository) AdminUpdateRole(ctx context.Context, projectRoleID int64, roleName, grade string, slotsTotal int32) (*domain.ProjectRole, error) {
	rows, err := r.projectRolesGate.Get(ctx, projectRoleFilter{ID: &projectRoleID})
	if err != nil || len(rows) == 0 {
		return nil, domain.ErrNotFound
	}
	updateSQL, updateArgs, err := psql.Update("project_roles").
		Set("role_name", roleName).
		Set("grade", grade).
		Set("slots_total", slotsTotal).
		Where(sq.Eq{"id": projectRoleID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build admin update role query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, updateSQL, updateArgs...); err != nil {
		return nil, fmt.Errorf("admin update role: %w", err)
	}
	roles, err := r.listProjectRoles(ctx, rows[0].ProjectID)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.ID == projectRoleID {
			return &role, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *ProjectRepository) AdminDeleteRole(ctx context.Context, projectRoleID int64) error {
	if err := r.projectRolesGate.Delete(ctx, projectRoleFilter{ID: &projectRoleID}); err != nil {
		if errors.Is(err, datagate.ErrNoRowsAffected) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("admin delete role: %w", err)
	}
	return nil
}

func (r *ProjectRepository) AdminDelete(ctx context.Context, projectID int64) error {
	selectSQL, selectArgs, err := psql.Select("pt.tag_id").
		From("project_tags pt").
		Join("tags t ON t.id = pt.tag_id").
		Where(sq.Eq{"pt.project_id": projectID}).
		Where(sq.Eq{"t.is_system": false}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build get custom tags query: %w", err)
	}

	var customTagIDs []int64
	rows, err := r.pool.Query(ctx, selectSQL, selectArgs...)
	if err == nil {
		for rows.Next() {
			var tid int64
			if rows.Scan(&tid) == nil {
				customTagIDs = append(customTagIDs, tid)
			}
		}
		rows.Close()
	}

	if err := r.projectsGate.Delete(ctx, projectFilter{ID: &projectID}); err != nil {
		if errors.Is(err, datagate.ErrNoRowsAffected) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("admin delete project: %w", err)
	}

	for _, tid := range customTagIDs {
		cleanSQL, cleanArgs, err := psql.Delete("tags").
			Where(sq.Eq{"id": tid}).
			Where(sq.Eq{"is_system": false}).
			Where(sq.Expr("NOT EXISTS (SELECT 1 FROM project_tags WHERE tag_id = ?)", tid)).
			ToSql()
		if err == nil {
			_, _ = r.pool.Exec(ctx, cleanSQL, cleanArgs...)
		}
	}

	return nil
}

func (r *ProjectRepository) DeleteRole(ctx context.Context, projectRoleID, requesterID int64) error {
	rows, err := r.projectRolesGate.Get(ctx, projectRoleFilter{ID: &projectRoleID})
	if err != nil || len(rows) == 0 {
		return domain.ErrNotFound
	}
	project, err := r.GetByID(ctx, rows[0].ProjectID)
	if err != nil {
		return err
	}
	if project.OwnerID != requesterID {
		return domain.ErrForbidden
	}
	if err := r.projectRolesGate.Delete(ctx, projectRoleFilter{ID: &projectRoleID}); err != nil {
		if errors.Is(err, datagate.ErrNoRowsAffected) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("delete project role: %w", err)
	}
	return nil
}

func (r *ProjectRepository) UpdateRole(ctx context.Context, projectRoleID, requesterID int64, roleName, grade string, slotsTotal int32) (*domain.ProjectRole, error) {
	rows, err := r.projectRolesGate.Get(ctx, projectRoleFilter{ID: &projectRoleID})
	if err != nil || len(rows) == 0 {
		return nil, domain.ErrNotFound
	}
	project, err := r.GetByID(ctx, rows[0].ProjectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerID != requesterID {
		return nil, domain.ErrForbidden
	}

	updateSQL, updateArgs, err := psql.Update("project_roles").
		Set("role_name", roleName).
		Set("grade", grade).
		Set("slots_total", slotsTotal).
		Where(sq.Eq{"id": projectRoleID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update project role query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, updateSQL, updateArgs...); err != nil {
		return nil, fmt.Errorf("update project role: %w", err)
	}

	roles, err := r.listProjectRoles(ctx, rows[0].ProjectID)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.ID == projectRoleID {
			return &role, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *ProjectRepository) AddRole(ctx context.Context, projectID, requesterID int64, roleName, grade string, slotsTotal int32) (*domain.ProjectRole, error) {
	project, err := r.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerID != requesterID {
		return nil, domain.ErrForbidden
	}

	projectRoleID, err := r.projectRolesGate.Create(ctx, projectRoleRow{
		ID:          0,
		ProjectID:   projectID,
		RoleName:    roleName,
		Grade:       grade,
		SlotsTotal:  slotsTotal,
		SlotsFilled: 0,
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("add role to project: %w", err)
	}

	roles, err := r.listProjectRoles(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, role := range roles {
		if role.ID == projectRoleID {
			return &role, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *ProjectRepository) ListByOwner(ctx context.Context, ownerID int64) ([]domain.Project, error) {
	query, args, err := psql.Select(
		"p.id", "p.owner_id", "p.title", "p.description", "p.status", "p.ban_reason", "p.appeal_status", "p.appeal_comment", "p.created_at",
		"u.id", "u.username", "u.email", "u.bio", "u.avatar_url", "u.created_at",
	).
		From("projects p").
		Join("users u ON u.id = p.owner_id").
		Where(sq.Eq{"p.owner_id": ownerID}).
		OrderBy("p.created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list by owner query: %w", err)
	}

	listRows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects by owner: %w", err)
	}
	defer listRows.Close()

	result := make([]domain.Project, 0)
	for listRows.Next() {
		var p domain.Project
		var u domain.User
		if err := listRows.Scan(
			&p.ID, &p.OwnerID, &p.Title, &p.Description, &p.Status, &p.BanReason, &p.AppealStatus, &p.AppealComment, &p.CreatedAt,
			&u.ID, &u.Username, &u.Email, &u.Bio, &u.AvatarURL, &u.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project row: %w", err)
		}
		p.Owner = &u
		result = append(result, p)
	}
	if err := listRows.Err(); err != nil {
		return nil, err
	}

	for i := range result {
		tags, err := r.listTagsByProjectID(ctx, result[i].ID)
		if err != nil {
			return nil, err
		}
		result[i].Tags = tags
	}

	return result, nil
}

func (r *ProjectRepository) AdminBan(ctx context.Context, projectID int64, reason string) (*domain.Project, error) {
	sql, args, err := psql.Update("projects").
		Set("status", domain.ProjectStatusBanned).
		Set("ban_reason", reason).
		Set("appeal_status", domain.AppealStatusNone).
		Where(sq.Eq{"id": projectID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build admin ban query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return nil, fmt.Errorf("admin ban project: %w", err)
	}
	return r.GetByID(ctx, projectID)
}

func (r *ProjectRepository) AdminUnban(ctx context.Context, projectID int64) (*domain.Project, error) {
	sql, args, err := psql.Update("projects").
		Set("status", domain.ProjectStatusDraft).
		Set("ban_reason", "").
		Set("appeal_status", domain.AppealStatusNone).
		Where(sq.Eq{"id": projectID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build admin unban query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return nil, fmt.Errorf("admin unban project: %w", err)
	}
	return r.GetByID(ctx, projectID)
}

func (r *ProjectRepository) SubmitAppeal(ctx context.Context, projectID, ownerID int64, comment string) (*domain.Project, error) {
	project, err := r.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerID != ownerID {
		return nil, domain.ErrForbidden
	}
	if project.Status != domain.ProjectStatusBanned {
		return nil, domain.NewError(400, "invalid_state", "project is not banned", nil)
	}
	if project.AppealStatus == domain.AppealStatusPending {
		return nil, domain.NewError(400, "invalid_state", "appeal already pending", nil)
	}
	sql, args, err := psql.Update("projects").
		Set("appeal_status", domain.AppealStatusPending).
		Set("appeal_comment", comment).
		Where(sq.Eq{"id": projectID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build submit appeal query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return nil, fmt.Errorf("submit appeal: %w", err)
	}
	return r.GetByID(ctx, projectID)
}

func (r *ProjectRepository) ReviewAppeal(ctx context.Context, projectID int64, approve bool, newBanReason string) (*domain.Project, error) {
	project, err := r.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.AppealStatus != domain.AppealStatusPending {
		return nil, domain.NewError(400, "invalid_state", "no pending appeal", nil)
	}
	var updateMap map[string]any
	if approve {
		updateMap = map[string]any{
			"status":         domain.ProjectStatusDraft,
			"ban_reason":     "",
			"appeal_status":  domain.AppealStatusApproved,
			"appeal_comment": "",
		}
	} else {
		updateMap = map[string]any{
			"appeal_status": domain.AppealStatusRejected,
		}
		if newBanReason != "" {
			updateMap["ban_reason"] = newBanReason
		}
	}
	b := psql.Update("projects").Where(sq.Eq{"id": projectID})
	for k, v := range updateMap {
		b = b.Set(k, v)
	}
	sql, args, err := b.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build review appeal query: %w", err)
	}
	if _, err := r.pool.Exec(ctx, sql, args...); err != nil {
		return nil, fmt.Errorf("review appeal: %w", err)
	}
	return r.GetByID(ctx, projectID)
}

func (r *ProjectRepository) listProjectRoles(ctx context.Context, projectID int64) ([]domain.ProjectRole, error) {
	query, args, err := psql.Select("id", "project_id", "role_name", "grade", "slots_total", "slots_filled").
		From("project_roles").
		Where(sq.Eq{"project_id": projectID}).
		OrderBy("id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list project roles query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list project roles: %w", err)
	}
	defer rows.Close()

	result := make([]domain.ProjectRole, 0)
	for rows.Next() {
		var pr domain.ProjectRole
		if err := rows.Scan(&pr.ID, &pr.ProjectID, &pr.RoleName, &pr.Grade, &pr.SlotsTotal, &pr.SlotsFilled); err != nil {
			return nil, fmt.Errorf("scan project role: %w", err)
		}
		result = append(result, pr)
	}
	return result, rows.Err()
}

func (r *ProjectRepository) listTagsByProjectID(ctx context.Context, projectID int64) ([]domain.Tag, error) {
	query, args, err := psql.Select("t.id", "t.name").
		From("project_tags pt").
		Join("tags t ON t.id = pt.tag_id").
		Where(sq.Eq{"pt.project_id": projectID}).
		OrderBy("t.name").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list project tags query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list project tags: %w", err)
	}
	defer rows.Close()

	tags := make([]domain.Tag, 0)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		tags = append(tags, domain.Tag{ID: id, Name: name})
	}
	return tags, rows.Err()
}
