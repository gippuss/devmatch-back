package service

import (
	"context"
	"testing"
	"time"

	"github.com/gippuss/devmatch-back/internal/auth"
	"github.com/gippuss/devmatch-back/internal/domain"
)

type userRepoMock struct {
	nextID       int64
	byID         map[int64]*domain.User
	byEmail      map[string]*domain.User
	passwordHash map[string]string
}

func newUserRepoMock() *userRepoMock {
	return &userRepoMock{
		nextID:       1,
		byID:         map[int64]*domain.User{},
		byEmail:      map[string]*domain.User{},
		passwordHash: map[string]string{},
	}
}

func (m *userRepoMock) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	user, ok := m.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (m *userRepoMock) GetAuthByEmail(_ context.Context, email string) (*domain.AuthUser, error) {
	user, ok := m.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &domain.AuthUser{
		ID:           user.ID,
		Email:        user.Email,
		Username:     user.Username,
		PasswordHash: m.passwordHash[email],
	}, nil
}

func (m *userRepoMock) GetByID(_ context.Context, userID int64) (*domain.User, error) {
	user, ok := m.byID[userID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (m *userRepoMock) CreateWithEmail(_ context.Context, email, passwordHash, username string) (*domain.User, error) {
	if _, exists := m.byEmail[email]; exists {
		return nil, domain.ErrConflict
	}
	user := &domain.User{
		ID:        m.nextID,
		Email:     email,
		Username:  username,
		Bio:       "",
		AvatarURL: "",
		Skills:    []domain.Skill{},
		CreatedAt: time.Now().UTC(),
	}
	m.byID[user.ID] = user
	m.byEmail[email] = user
	m.passwordHash[email] = passwordHash
	m.nextID++
	return user, nil
}

func (m *userRepoMock) SetPasswordHash(_ context.Context, userID int64, passwordHash string) error {
	user, ok := m.byID[userID]
	if !ok {
		return domain.ErrNotFound
	}
	m.passwordHash[user.Email] = passwordHash
	return nil
}

func (m *userRepoMock) UpdateProfile(_ context.Context, userID int64, username, bio, avatarURL *string, _ []int64) (*domain.User, error) {
	user, ok := m.byID[userID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if username != nil {
		user.Username = *username
	}
	if bio != nil {
		user.Bio = *bio
	}
	if avatarURL != nil {
		user.AvatarURL = *avatarURL
	}
	return user, nil
}

type refreshRepoMock struct {
	store map[string]refreshTokenState
}

type refreshTokenState struct {
	userID    int64
	rawToken  string
	expiresAt time.Time
	revoked   bool
}

func newRefreshRepoMock() *refreshRepoMock {
	return &refreshRepoMock{store: map[string]refreshTokenState{}}
}

func (m *refreshRepoMock) Store(_ context.Context, userID int64, jti string, rawToken string, expiresAt time.Time) error {
	m.store[jti] = refreshTokenState{userID: userID, rawToken: rawToken, expiresAt: expiresAt}
	return nil
}

func (m *refreshRepoMock) ValidateActive(_ context.Context, jti string, rawToken string) (*domain.StoredRefreshToken, error) {
	state, ok := m.store[jti]
	if !ok || state.revoked || state.rawToken != rawToken || time.Now().UTC().After(state.expiresAt) {
		return nil, domain.ErrUnauthorized
	}
	return &domain.StoredRefreshToken{UserID: state.userID, JTI: jti, ExpiresAt: state.expiresAt}, nil
}

func (m *refreshRepoMock) RevokeByJTI(_ context.Context, jti string) error {
	state, ok := m.store[jti]
	if !ok {
		return domain.ErrNotFound
	}
	state.revoked = true
	m.store[jti] = state
	return nil
}

func newAuthServiceForTest(t *testing.T) *AuthService {
	t.Helper()
	manager, err := auth.NewTokenManager(auth.Config{
		AccessSecret:  "access_test_secret",
		RefreshSecret: "refresh_test_secret",
		AccessTTL:     10 * time.Minute,
		RefreshTTL:    time.Hour,
	})
	if err != nil {
		t.Fatalf("failed to init token manager: %v", err)
	}
	return NewAuthService(newUserRepoMock(), newRefreshRepoMock(), manager)
}

func TestAuthServiceRegisterAndLogin(t *testing.T) {
	svc := newAuthServiceForTest(t)

	registerRes, err := svc.Register(context.Background(), RegisterInput{
		Email:    "demo@example.com",
		Username: "demo",
		Password: "strongpassword123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if registerRes.Token.AccessToken == "" || registerRes.Token.RefreshToken == "" {
		t.Fatal("expected tokens in register response")
	}

	loginRes, err := svc.Login(context.Background(), LoginInput{
		Email:    "demo@example.com",
		Password: "strongpassword123",
	})
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if loginRes.User.Email != "demo@example.com" {
		t.Fatalf("unexpected user email: %s", loginRes.User.Email)
	}
}

func TestAuthServiceLoginInvalidPassword(t *testing.T) {
	svc := newAuthServiceForTest(t)
	_, err := svc.Register(context.Background(), RegisterInput{
		Email:    "demo@example.com",
		Username: "demo",
		Password: "strongpassword123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err = svc.Login(context.Background(), LoginInput{
		Email:    "demo@example.com",
		Password: "wrong-password",
	})
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	if appErr := domain.AsAppError(err); appErr.Code != domain.ErrUnauthorized.Code {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthServiceRefreshRotatesRefreshToken(t *testing.T) {
	svc := newAuthServiceForTest(t)
	registerRes, err := svc.Register(context.Background(), RegisterInput{
		Email:    "demo@example.com",
		Username: "demo",
		Password: "strongpassword123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	refreshed, err := svc.Refresh(context.Background(), registerRes.Token.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshed.Token.RefreshToken == registerRes.Token.RefreshToken {
		t.Fatal("expected refresh rotation")
	}

	_, err = svc.Refresh(context.Background(), registerRes.Token.RefreshToken)
	if err == nil {
		t.Fatal("expected old refresh token to be revoked")
	}
}

func TestAuthServiceLogoutRevokesRefreshToken(t *testing.T) {
	svc := newAuthServiceForTest(t)
	registerRes, err := svc.Register(context.Background(), RegisterInput{
		Email:    "demo@example.com",
		Username: "demo",
		Password: "strongpassword123",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	if err := svc.Logout(context.Background(), registerRes.Token.RefreshToken); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	_, err = svc.Refresh(context.Background(), registerRes.Token.RefreshToken)
	if err == nil {
		t.Fatal("expected revoked refresh token to fail")
	}
}
