package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/gippuss/devmatch-back/internal/auth"
	"github.com/gippuss/devmatch-back/internal/domain"
)

type AuthService struct {
	users         UserRepository
	refreshTokens RefreshTokenRepository
	tokens        *auth.TokenManager
}

type RegisterInput struct {
	Email    string
	Username string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	User  *domain.User    `json:"user"`
	Token *auth.TokenPair `json:"token"`
}

func NewAuthService(users UserRepository, refreshTokens RefreshTokenRepository, tokens *auth.TokenManager) *AuthService {
	return &AuthService{users: users, refreshTokens: refreshTokens, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	email := normalizeEmail(input.Email)
	username := strings.TrimSpace(input.Username)
	if email == "" || username == "" || len(input.Password) < 8 {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "email, username and password(>=8) are required", nil)
	}

	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, domain.NewError(domain.ErrConflict.Status, domain.ErrConflict.Code, "email already exists", nil)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, domain.Wrap(domain.ErrInternal, err, "failed to hash password")
	}

	user, err := s.users.CreateWithEmail(ctx, email, string(hash), username)
	if err != nil {
		return nil, err
	}

	return s.issueSession(ctx, user.ID)
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	email := normalizeEmail(input.Email)
	if email == "" || input.Password == "" {
		return nil, domain.NewError(domain.ErrValidation.Status, domain.ErrValidation.Code, "email and password are required", nil)
	}

	authUser, err := s.users.GetAuthByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewError(domain.ErrUnauthorized.Status, domain.ErrUnauthorized.Code, "invalid credentials", nil)
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(authUser.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.NewError(domain.ErrUnauthorized.Status, domain.ErrUnauthorized.Code, "invalid credentials", nil)
	}

	return s.issueSession(ctx, authUser.ID)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := s.tokens.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, domain.NewError(domain.ErrUnauthorized.Status, domain.ErrUnauthorized.Code, "invalid refresh token", err)
	}

	stored, err := s.refreshTokens.ValidateActive(ctx, claims.JTI, refreshToken)
	if err != nil {
		return nil, domain.NewError(domain.ErrUnauthorized.Status, domain.ErrUnauthorized.Code, "invalid refresh token", err)
	}

	if err := s.refreshTokens.RevokeByJTI(ctx, stored.JTI); err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	return s.issueSession(ctx, stored.UserID)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.tokens.ParseRefreshToken(refreshToken)
	if err != nil {
		return domain.NewError(domain.ErrUnauthorized.Status, domain.ErrUnauthorized.Code, "invalid refresh token", err)
	}

	if err := s.refreshTokens.RevokeByJTI(ctx, claims.JTI); err != nil && !errors.Is(err, domain.ErrNotFound) {
		return err
	}

	return nil
}

func (s *AuthService) issueSession(ctx context.Context, userID int64) (*AuthResult, error) {
	jti := uuid.NewString()
	tokenPair, err := s.tokens.IssueTokenPair(userID, jti)
	if err != nil {
		return nil, domain.Wrap(domain.ErrInternal, err, "failed to issue token")
	}

	if err := s.refreshTokens.Store(ctx, userID, jti, tokenPair.RefreshToken, tokenPair.RefreshExpiresAt); err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &AuthResult{User: user, Token: tokenPair}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
