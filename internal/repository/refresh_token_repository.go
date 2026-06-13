package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/gippuss/datagate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gippuss/devmatch-back/internal/domain"
)

type RefreshTokenRepository struct {
	pool *pgxpool.Pool
	gate datagate.DataGate[refreshTokenRow, refreshTokenFilter]
}

type refreshTokenRow struct {
	ID        int64      `db:"id"`
	UserID    int64      `db:"user_id" insert:"user_id"`
	JTI       string     `db:"jti" insert:"jti"`
	TokenHash string     `db:"token_hash" insert:"token_hash"`
	ExpiresAt time.Time  `db:"expires_at" insert:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at" insert:"revoked_at"`
	CreatedAt time.Time  `db:"created_at" insert:"created_at"`
}

type refreshTokenFilter struct {
	JTI *string `filter:"jti"`
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) (*RefreshTokenRepository, error) {
	gate, err := datagate.NewDataGate[refreshTokenRow, refreshTokenFilter]("refresh_tokens", "id", pool)
	if err != nil {
		return nil, err
	}

	return &RefreshTokenRepository{pool: pool, gate: gate}, nil
}

func (r *RefreshTokenRepository) Store(ctx context.Context, userID int64, jti string, rawToken string, expiresAt time.Time) error {
	_, err := r.gate.Create(ctx, refreshTokenRow{
		ID:        0,
		UserID:    userID,
		JTI:       jti,
		TokenHash: hashToken(rawToken),
		ExpiresAt: expiresAt,
		RevokedAt: nil,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("store refresh token: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepository) ValidateActive(ctx context.Context, jti string, rawToken string) (*domain.StoredRefreshToken, error) {
	rows, err := r.gate.Get(ctx, refreshTokenFilter{JTI: &jti})
	if err != nil {
		return nil, fmt.Errorf("get refresh token by jti: %w", err)
	}
	if len(rows) == 0 {
		return nil, domain.ErrUnauthorized
	}

	row := rows[0]
	if row.RevokedAt != nil || time.Now().UTC().After(row.ExpiresAt) {
		return nil, domain.ErrUnauthorized
	}
	if row.TokenHash != hashToken(rawToken) {
		return nil, domain.ErrUnauthorized
	}

	return &domain.StoredRefreshToken{
		UserID:    row.UserID,
		JTI:       row.JTI,
		ExpiresAt: row.ExpiresAt,
		RevokedAt: row.RevokedAt,
	}, nil
}

func (r *RefreshTokenRepository) RevokeByJTI(ctx context.Context, jti string) error {
	now := time.Now().UTC()
	if err := r.gate.Update(ctx, refreshTokenFilter{JTI: &jti}, map[string]interface{}{"revoked_at": &now}); err != nil {
		if errors.Is(err, datagate.ErrNoRowsAffected) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
