package repository

import (
	"context"
	"fmt"

	"github.com/gippuss/datagate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gippuss/devmatch-back/internal/domain"
)

type DictionaryRepository struct {
	tagsGate   datagate.DataGate[tagRow, noFilter]
	skillsGate datagate.DataGate[skillRow, noFilter]
	pool       *pgxpool.Pool
}

type tagRow struct {
	ID       int64  `db:"id"`
	Name     string `db:"name" insert:"name"`
	IsSystem bool   `db:"is_system"`
}

type skillRow struct {
	ID   int64  `db:"id"`
	Name string `db:"name" insert:"name"`
}

type noFilter struct{}

func NewDictionaryRepository(pool *pgxpool.Pool) (*DictionaryRepository, error) {
	tagsGate, err := datagate.NewDataGate[tagRow, noFilter]("tags", "id", pool)
	if err != nil {
		return nil, err
	}

	skillsGate, err := datagate.NewDataGate[skillRow, noFilter]("skills", "id", pool)
	if err != nil {
		return nil, err
	}

	return &DictionaryRepository{tagsGate: tagsGate, skillsGate: skillsGate, pool: pool}, nil
}

func (r *DictionaryRepository) ListTags(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.tagsGate.Get(ctx, noFilter{})
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	result := make([]domain.Tag, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.Tag{ID: row.ID, Name: row.Name, IsSystem: row.IsSystem})
	}
	return result, nil
}

func (r *DictionaryRepository) ListSystemTags(ctx context.Context) ([]domain.Tag, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, is_system FROM tags WHERE is_system = TRUE ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list system tags: %w", err)
	}
	defer rows.Close()

	result := make([]domain.Tag, 0)
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.IsSystem); err != nil {
			return nil, fmt.Errorf("scan system tag: %w", err)
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

func (r *DictionaryRepository) ListSkills(ctx context.Context) ([]domain.Skill, error) {
	rows, err := r.skillsGate.Get(ctx, noFilter{})
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	result := make([]domain.Skill, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.Skill{ID: row.ID, Name: row.Name})
	}
	return result, nil
}

func (r *DictionaryRepository) CreateTag(ctx context.Context, name string) (*domain.Tag, error) {
	var tag domain.Tag
	err := r.pool.QueryRow(ctx,
		`INSERT INTO tags (name, is_system) VALUES ($1, FALSE)
		 ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		 RETURNING id, name, is_system`,
		name,
	).Scan(&tag.ID, &tag.Name, &tag.IsSystem)
	if err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}
	return &tag, nil
}

func (r *DictionaryRepository) DeleteTagIfUnused(ctx context.Context, tagID int64) error {
	// Only delete custom (non-system) tags that are no longer used by any project
	_, err := r.pool.Exec(ctx, `
		DELETE FROM tags
		WHERE id = $1
		  AND is_system = FALSE
		  AND NOT EXISTS (SELECT 1 FROM project_tags WHERE tag_id = $1)
	`, tagID)
	if err != nil {
		return fmt.Errorf("delete unused custom tag: %w", err)
	}
	return nil
}
