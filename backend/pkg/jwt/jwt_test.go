package jwt

import (
	"testing"
	"time"

	"echo-union/backend/config"
)

func newTestManager() *Manager {
	return NewManager(&config.AuthConfig{
		JWTSecret:               "test-secret-key-for-unit-testing-2026",
		AccessTokenTTL:          15 * time.Minute,
		RefreshTokenTTLDefault:  24 * time.Hour,
		RefreshTokenTTLRemember: 7 * 24 * time.Hour,
	})
}

func TestGenerateAndParseAccessToken(t *testing.T) {
	m := newTestManager()

	token, err := m.GenerateAccessToken("user-1", "admin", "dept-1")
	if err != nil {
		t.Fatalf("GenerateAccessToken 失败: %v", err)
	}

	claims, err := m.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken 失败: %v", err)
	}

	if claims.UserID != "user-1" {
		t.Errorf("期望 UserID=user-1，实际=%s", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("期望 Role=admin，实际=%s", claims.Role)
	}
	if claims.DepartmentID != "dept-1" {
		t.Errorf("期望 DepartmentID=dept-1，实际=%s", claims.DepartmentID)
	}
	if claims.TokenType != "access" {
		t.Errorf("期望 TokenType=access，实际=%s", claims.TokenType)
	}
	if claims.Issuer != "echo-union" {
		t.Errorf("期望 Issuer=echo-union，实际=%s", claims.Issuer)
	}
	if claims.ID == "" {
		t.Error("JTI 不应为空")
	}
}

func TestGenerateRefreshToken_Default(t *testing.T) {
	m := newTestManager()

	token, err := m.GenerateRefreshToken("user-1", "member", "dept-1", false)
	if err != nil {
		t.Fatalf("GenerateRefreshToken 失败: %v", err)
	}

	claims, err := m.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken 失败: %v", err)
	}

	if claims.TokenType != "refresh" {
		t.Errorf("期望 TokenType=refresh，实际=%s", claims.TokenType)
	}
	if claims.RememberMe != false {
		t.Error("期望 RememberMe=false")
	}

	// 检查过期时间约为 24h
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl < 23*time.Hour || ttl > 25*time.Hour {
		t.Errorf("默认 RefreshToken TTL 期望约24h，实际=%v", ttl)
	}
}

func TestGenerateRefreshToken_RememberMe(t *testing.T) {
	m := newTestManager()

	token, err := m.GenerateRefreshToken("user-1", "member", "dept-1", true)
	if err != nil {
		t.Fatalf("GenerateRefreshToken(RememberMe) 失败: %v", err)
	}

	claims, err := m.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken 失败: %v", err)
	}

	if claims.RememberMe != true {
		t.Error("期望 RememberMe=true")
	}

	// 检查过期时间约为 7 天
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl < 6*24*time.Hour || ttl > 8*24*time.Hour {
		t.Errorf("RememberMe RefreshToken TTL 期望约7天，实际=%v", ttl)
	}
}

func TestParseToken_InvalidToken(t *testing.T) {
	m := newTestManager()

	_, err := m.ParseToken("invalid.token.string")
	if err == nil {
		t.Error("期望解析无效 token 返回错误")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	m1 := newTestManager()
	m2 := NewManager(&config.AuthConfig{
		JWTSecret:      "different-secret-key",
		AccessTokenTTL: 15 * time.Minute,
	})

	token, _ := m1.GenerateAccessToken("user-1", "admin", "dept-1")
	_, err := m2.ParseToken(token)
	if err == nil {
		t.Error("不同密钥签名的 token 不应通过验证")
	}
}

func TestParseToken_ExpiredToken(t *testing.T) {
	// 创建一个 TTL 极短的 manager 来测试过期
	m := NewManager(&config.AuthConfig{
		JWTSecret:              "test-secret",
		AccessTokenTTL:         1 * time.Millisecond,
		RefreshTokenTTLDefault: 1 * time.Millisecond,
	})

	token, _ := m.GenerateAccessToken("user-1", "admin", "dept-1")
	time.Sleep(10 * time.Millisecond)

	_, err := m.ParseToken(token)
	if err == nil {
		t.Error("过期 token 不应通过验证")
	}
	if err != ErrTokenExpired {
		t.Errorf("期望 ErrTokenExpired，实际: %v", err)
	}
}
