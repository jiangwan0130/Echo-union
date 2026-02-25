package jwt

import (
	"errors"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"echo-union/backend/config"
)

var (
	ErrTokenExpired = errors.New("token 已过期")
	ErrTokenInvalid = errors.New("token 无效")
)

// Claims 自定义 JWT 声明
type Claims struct {
	UserID       string `json:"user_id"`
	Role         string `json:"role"`
	DepartmentID string `json:"department_id"`
	TokenType    string `json:"token_type"`            // "access" | "refresh"
	RememberMe   bool   `json:"remember_me,omitempty"` // 仅 refresh token 使用
	jwtv5.RegisteredClaims
}

// Manager JWT 管理器
type Manager struct {
	secret                  []byte
	accessTokenTTL          time.Duration
	refreshTokenTTLDefault  time.Duration
	refreshTokenTTLRemember time.Duration
}

// NewManager 创建 JWT 管理器
func NewManager(cfg *config.AuthConfig) *Manager {
	return &Manager{
		secret:                  []byte(cfg.JWTSecret),
		accessTokenTTL:          cfg.AccessTokenTTL,
		refreshTokenTTLDefault:  cfg.RefreshTokenTTLDefault,
		refreshTokenTTLRemember: cfg.RefreshTokenTTLRemember,
	}
}

// GenerateAccessToken 生成 Access Token
func (m *Manager) GenerateAccessToken(userID, role, departmentID string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:       userID,
		Role:         role,
		DepartmentID: departmentID,
		TokenType:    "access",
		RegisteredClaims: jwtv5.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(m.accessTokenTTL)),
			Issuer:    "echo-union",
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// GenerateRefreshToken 生成 Refresh Token
// rememberMe 为 true 时使用更长的有效期
func (m *Manager) GenerateRefreshToken(userID, role, departmentID string, rememberMe bool) (string, error) {
	ttl := m.refreshTokenTTLDefault
	if rememberMe {
		ttl = m.refreshTokenTTLRemember
	}

	now := time.Now()
	claims := Claims{
		UserID:       userID,
		Role:         role,
		DepartmentID: departmentID,
		TokenType:    "refresh",
		RememberMe:   rememberMe,
		RegisteredClaims: jwtv5.RegisteredClaims{
			ID:        uuid.New().String(),
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(ttl)),
			Issuer:    "echo-union",
		},
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseToken 解析并验证 Token
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwtv5.ParseWithClaims(tokenString, &Claims{}, func(t *jwtv5.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, ErrTokenInvalid
		}
		return m.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwtv5.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// [自证通过] pkg/jwt/jwt.go
