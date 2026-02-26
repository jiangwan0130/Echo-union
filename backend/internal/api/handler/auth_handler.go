package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"echo-union/backend/config"
	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

// AuthHandler 认证模块 HTTP 处理器
type AuthHandler struct {
	authSvc   service.AuthService
	cookieCfg *config.CookieConfig
}

// NewAuthHandler 创建 AuthHandler
func NewAuthHandler(authSvc service.AuthService, cookieCfg *config.CookieConfig) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, cookieCfg: cookieCfg}
}

// ────────────────────── Login ──────────────────────
// POST /api/v1/auth/login

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	result, err := h.authSvc.Login(c.Request.Context(), &req)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	// 双模式：Set-Cookie + JSON body
	h.setRefreshTokenCookie(c, result.RefreshToken)
	response.OK(c, result)
}

// ────────────────────── Logout ──────────────────────
// POST /api/v1/auth/logout

func (h *AuthHandler) Logout(c *gin.Context) {
	// 从中间件注入的上下文中获取 JTI 和过期时间
	jti, _ := c.Get("token_jti")
	expRaw, _ := c.Get("token_exp")

	var accessJTI string
	var accessExp time.Time
	if j, ok := jti.(string); ok {
		accessJTI = j
	}
	if e, ok := expRaw.(time.Time); ok {
		accessExp = e
	}

	// 从 Cookie 或请求体获取 refresh token
	refreshToken := h.extractRefreshToken(c)

	if err := h.authSvc.Logout(c.Request.Context(), accessJTI, accessExp, refreshToken); err != nil {
		response.InternalError(c)
		return
	}

	// 清除 refresh token Cookie
	h.clearRefreshTokenCookie(c)
	response.OK(c, nil)
}

// ────────────────────── RefreshToken ──────────────────────
// POST /api/v1/auth/refresh

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// 优先从 Cookie 读取，回退到请求体
	refreshToken := h.extractRefreshToken(c)
	if refreshToken == "" {
		response.BadRequest(c, 10001, "缺少 refresh_token")
		return
	}

	result, err := h.authSvc.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	// 双模式：写入新的 Cookie
	h.setRefreshTokenCookie(c, result.RefreshToken)
	response.OK(c, result)
}

// ────────────────────── GetCurrentUser ──────────────────────
// GET /api/v1/auth/me

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	result, err := h.authSvc.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			response.NotFound(c, 12001, "用户不存在")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, result)
}

// ────────────────────── ChangePassword ──────────────────────
// PUT /api/v1/auth/password

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, ok := MustGetUserID(c)
	if !ok {
		return
	}

	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, 10001, "参数校验失败")
		return
	}

	if err := h.authSvc.ChangePassword(c.Request.Context(), userID, &req); err != nil {
		h.handleAuthError(c, err)
		return
	}

	response.OK(c, nil)
}

// ── 内部辅助方法 ──

// handleAuthError 统一处理认证模块业务错误到 HTTP 响应
func (h *AuthHandler) handleAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, 11001, "学号或密码错误")
	case errors.Is(err, service.ErrTokenExpired):
		response.Error(c, http.StatusUnauthorized, 11002, "Token已过期")
	case errors.Is(err, service.ErrTokenInvalid):
		response.Error(c, http.StatusUnauthorized, 11003, "Token无效")
	case errors.Is(err, service.ErrTokenBlacklisted):
		response.Error(c, http.StatusUnauthorized, 11003, "Token已被吊销")
	case errors.Is(err, service.ErrEmailExists):
		response.BadRequest(c, 11005, "邮箱已被注册")
	case errors.Is(err, service.ErrStudentIDExists):
		response.BadRequest(c, 11006, "学号已被注册")
	case errors.Is(err, service.ErrWeakPassword):
		response.BadRequest(c, 10001, err.Error())
	case errors.Is(err, service.ErrOldPasswordWrong):
		response.Error(c, http.StatusUnauthorized, 11001, "原密码错误")
	case errors.Is(err, service.ErrUserNotFound):
		response.NotFound(c, 12001, "用户不存在")
	default:
		response.InternalError(c)
	}
}

// extractRefreshToken 从 Cookie 或请求体中提取 Refresh Token
func (h *AuthHandler) extractRefreshToken(c *gin.Context) string {
	// 优先 Cookie
	if token, err := c.Cookie("refresh_token"); err == nil && token != "" {
		return token
	}

	// 回退请求体
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		return req.RefreshToken
	}

	return ""
}

// setRefreshTokenCookie 设置 HttpOnly Refresh Token Cookie
func (h *AuthHandler) setRefreshTokenCookie(c *gin.Context, token string) {
	sameSite := http.SameSiteLaxMode
	if h.cookieCfg != nil && h.cookieCfg.SameSite == "Strict" {
		sameSite = http.SameSiteStrictMode
	}

	secure := false
	domain := ""
	if h.cookieCfg != nil {
		secure = h.cookieCfg.Secure
		domain = h.cookieCfg.Domain
	}

	c.SetSameSite(sameSite)
	c.SetCookie(
		"refresh_token",
		token,
		7*24*3600, // max-age 秒：与最长 remember_me TTL 对齐
		"/api/v1/auth",
		domain,
		secure,
		true, // HttpOnly
	)
}

// clearRefreshTokenCookie 清除 Refresh Token Cookie
func (h *AuthHandler) clearRefreshTokenCookie(c *gin.Context) {
	domain := ""
	secure := false
	if h.cookieCfg != nil {
		domain = h.cookieCfg.Domain
		secure = h.cookieCfg.Secure
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(
		"refresh_token",
		"",
		-1,
		"/api/v1/auth",
		domain,
		secure,
		true,
	)
}
