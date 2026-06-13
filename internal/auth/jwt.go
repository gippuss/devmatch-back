package auth

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type TokenManager struct {
	cfg Config
}

type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

type TokenClaims struct {
	UserID int64
	JTI    string
	Type   string
}

func NewTokenManager(cfg Config) (*TokenManager, error) {
	if cfg.AccessSecret == "" || cfg.RefreshSecret == "" {
		return nil, fmt.Errorf("auth secrets are required")
	}
	if cfg.AccessTTL <= 0 || cfg.RefreshTTL <= 0 {
		return nil, fmt.Errorf("auth TTL must be positive")
	}

	return &TokenManager{cfg: cfg}, nil
}

func (m *TokenManager) IssueTokenPair(userID int64, refreshJTI string) (*TokenPair, error) {
	now := time.Now().UTC()
	accessExp := now.Add(m.cfg.AccessTTL)
	refreshExp := now.Add(m.cfg.RefreshTTL)

	accessToken, err := m.signToken(m.cfg.AccessSecret, userID, "access", "", accessExp)
	if err != nil {
		return nil, err
	}
	refreshToken, err := m.signToken(m.cfg.RefreshSecret, userID, "refresh", refreshJTI, refreshExp)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func (m *TokenManager) ParseAccessToken(raw string) (*TokenClaims, error) {
	return m.parseToken(raw, m.cfg.AccessSecret, "access")
}

func (m *TokenManager) ParseRefreshToken(raw string) (*TokenClaims, error) {
	return m.parseToken(raw, m.cfg.RefreshSecret, "refresh")
}

func (m *TokenManager) signToken(secret string, userID int64, tokenType string, jti string, exp time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":  strconv.FormatInt(userID, 10),
		"type": tokenType,
		"exp":  exp.Unix(),
		"iat":  time.Now().UTC().Unix(),
	}
	if jti != "" {
		claims["jti"] = jti
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (m *TokenManager) parseToken(raw, secret, expectedType string) (*TokenClaims, error) {
	token, err := jwt.Parse(raw, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}
	tokenType, _ := claims["type"].(string)
	if tokenType != expectedType {
		return nil, fmt.Errorf("invalid token type")
	}
	sub, _ := claims["sub"].(string)
	userID, err := strconv.ParseInt(sub, 10, 64)
	if err != nil || userID <= 0 {
		return nil, fmt.Errorf("invalid sub claim")
	}
	jti, _ := claims["jti"].(string)
	return &TokenClaims{UserID: userID, JTI: jti, Type: tokenType}, nil
}
