package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gippuss/datagate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gippuss/devmatch-back/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
	gate datagate.DataGate[userRow, userFilter]
}

type userRow struct {
	ID           int64     `db:"id"`
	Email        string    `db:"email" insert:"email"`
	PasswordHash string    `db:"password_hash" insert:"password_hash"`
	Username     string    `db:"username" insert:"username"`
	Bio          string    `db:"bio" insert:"bio"`
	AvatarURL    string    `db:"avatar_url" insert:"avatar_url"`
	IsAdmin      bool      `db:"is_admin" insert:"is_admin"`
	CreatedAt    time.Time `db:"created_at" insert:"created_at"`
}

type userFilter struct {
	ID    *int64  `filter:"id"`
	Email *string `filter:"email"`
}

func NewUserRepository(pool *pgxpool.Pool) (*UserRepository, error) {
	gate, err := datagate.NewDataGate[userRow, userFilter]("users", "id", pool)
	if err != nil {
		return nil, err
	}

	return &UserRepository{pool: pool, gate: gate}, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	rows, err := r.gate.Get(ctx, userFilter{Email: &email})
	if err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}

	user := rowToUser(rows[0])
	skills, err := r.listSkillsByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Skills = skills

	return user, nil
}

func (r *UserRepository) GetAuthByEmail(ctx context.Context, email string) (*domain.AuthUser, error) {
	rows, err := r.gate.Get(ctx, userFilter{Email: &email})
	if err != nil {
		return nil, fmt.Errorf("get auth user by email: %w", err)
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}

	return &domain.AuthUser{
		ID:           rows[0].ID,
		Email:        rows[0].Email,
		PasswordHash: rows[0].PasswordHash,
		Username:     rows[0].Username,
	}, nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID int64) (*domain.User, error) {
	rows, err := r.gate.Get(ctx, userFilter{ID: &userID})
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}

	user := rowToUser(rows[0])
	skills, err := r.listSkillsByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Skills = skills

	return user, nil
}

func (r *UserRepository) CreateWithEmail(ctx context.Context, email, passwordHash, username string) (*domain.User, error) {
	data := userRow{
		ID:           0,
		Email:        email,
		PasswordHash: passwordHash,
		Username:     username,
		Bio:          "",
		AvatarURL:    "",
		CreatedAt:    time.Now().UTC(),
	}

	id, err := r.gate.Create(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *UserRepository) SetPasswordHash(ctx context.Context, userID int64, passwordHash string) error {
	if err := r.gate.Update(ctx, userFilter{ID: &userID}, map[string]interface{}{"password_hash": passwordHash}); err != nil {
		if errors.Is(err, datagate.ErrNoRowsAffected) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("set password hash: %w", err)
	}

	return nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, userID int64, username, bio, avatarURL *string, skillIDs []int64) (*domain.User, error) {
	updateData := map[string]interface{}{}
	if username != nil {
		updateData["username"] = *username
	}
	if bio != nil {
		updateData["bio"] = *bio
	}
	if avatarURL != nil {
		updateData["avatar_url"] = *avatarURL
	}

	if len(updateData) > 0 {
		if err := r.gate.Update(ctx, userFilter{ID: &userID}, updateData); err != nil {
			if errors.Is(err, datagate.ErrNoRowsAffected) {
				return nil, domain.ErrNotFound
			}
			return nil, fmt.Errorf("update user profile: %w", err)
		}
	}

	if skillIDs != nil {
		tx, err := r.pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin tx for user skills update: %w", err)
		}
		defer tx.Rollback(ctx)

		if _, err := tx.Exec(ctx, "DELETE FROM user_skills WHERE user_id = $1", userID); err != nil {
			return nil, fmt.Errorf("clear user skills: %w", err)
		}

		for _, skillID := range skillIDs {
			if _, err := tx.Exec(ctx, "INSERT INTO user_skills (user_id, skill_id) VALUES ($1, $2)", userID, skillID); err != nil {
				return nil, fmt.Errorf("insert user skill: %w", err)
			}
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit user skills update: %w", err)
		}
	}

	return r.GetByID(ctx, userID)
}

func (r *UserRepository) listSkillsByUserID(ctx context.Context, userID int64) ([]domain.Skill, error) {
	query := `
SELECT s.id, s.name
FROM skills s
JOIN user_skills us ON us.skill_id = s.id
WHERE us.user_id = $1
ORDER BY s.name
`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list user skills: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Skill, 0)
	for rows.Next() {
		var id int64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan skill row: %w", err)
		}
		result = append(result, domain.Skill{ID: id, Name: name})
	}

	return result, rows.Err()
}

func rowToUser(row userRow) *domain.User {
	return &domain.User{
		ID:        row.ID,
		Email:     row.Email,
		Username:  row.Username,
		Bio:       row.Bio,
		AvatarURL: row.AvatarURL,
		IsAdmin:   row.IsAdmin,
		CreatedAt: row.CreatedAt,
		Skills:    []domain.Skill{},
	}
}
