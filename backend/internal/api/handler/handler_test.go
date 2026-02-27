package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"echo-union/backend/internal/dto"
	"echo-union/backend/internal/service"
	"echo-union/backend/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ═══════════════════════════════════════════════════════════
// Mock Services
// ═══════════════════════════════════════════════════════════

// ── Mock AuthService ──

type mockAuthService struct {
	loginResult      *dto.TokenResponse
	loginErr         error
	refreshResult    *dto.TokenResponse
	refreshErr       error
	logoutErr        error
	getCurrentResult *dto.UserDetailResponse
	getCurrentErr    error
	changePassErr    error
}

func (m *mockAuthService) Login(_ context.Context, _ *dto.LoginRequest) (*dto.TokenResponse, error) {
	return m.loginResult, m.loginErr
}
func (m *mockAuthService) RefreshToken(_ context.Context, _ string) (*dto.TokenResponse, error) {
	return m.refreshResult, m.refreshErr
}
func (m *mockAuthService) Logout(_ context.Context, _ string, _ time.Time, _ string) error {
	return m.logoutErr
}
func (m *mockAuthService) GetCurrentUser(_ context.Context, _ string) (*dto.UserDetailResponse, error) {
	return m.getCurrentResult, m.getCurrentErr
}
func (m *mockAuthService) ChangePassword(_ context.Context, _ string, _ *dto.ChangePasswordRequest) error {
	return m.changePassErr
}

// ── Mock ScheduleService ──

type mockScheduleService struct {
	autoResult            *dto.AutoScheduleResponse
	autoErr               error
	getResult             *dto.ScheduleResponse
	getErr                error
	myResult              *dto.ScheduleResponse
	myErr                 error
	updateItemResult      *dto.ScheduleItemResponse
	updateItemErr         error
	validateResult        *dto.ValidateCandidateResponse
	validateErr           error
	candidatesResult      []dto.CandidateResponse
	candidatesErr         error
	publishResult         *dto.ScheduleResponse
	publishErr            error
	updatePublishedResult *dto.ScheduleItemResponse
	updatePublishedErr    error
	changeLogsResult      []dto.ScheduleChangeLogResponse
	changeLogsTotal       int64
	changeLogsErr         error
	scopeResult           *dto.ScopeCheckResponse
	scopeErr              error
}

func (m *mockScheduleService) AutoSchedule(_ context.Context, _ *dto.AutoScheduleRequest, _ string) (*dto.AutoScheduleResponse, error) {
	return m.autoResult, m.autoErr
}
func (m *mockScheduleService) GetSchedule(_ context.Context, _ string) (*dto.ScheduleResponse, error) {
	return m.getResult, m.getErr
}
func (m *mockScheduleService) GetMySchedule(_ context.Context, _, _ string) (*dto.ScheduleResponse, error) {
	return m.myResult, m.myErr
}
func (m *mockScheduleService) UpdateItem(_ context.Context, _ string, _ *dto.UpdateScheduleItemRequest, _ string) (*dto.ScheduleItemResponse, error) {
	return m.updateItemResult, m.updateItemErr
}
func (m *mockScheduleService) ValidateCandidate(_ context.Context, _ string, _ *dto.ValidateCandidateRequest) (*dto.ValidateCandidateResponse, error) {
	return m.validateResult, m.validateErr
}
func (m *mockScheduleService) GetCandidates(_ context.Context, _ string) ([]dto.CandidateResponse, error) {
	return m.candidatesResult, m.candidatesErr
}
func (m *mockScheduleService) Publish(_ context.Context, _ *dto.PublishScheduleRequest, _ string) (*dto.ScheduleResponse, error) {
	return m.publishResult, m.publishErr
}
func (m *mockScheduleService) UpdatePublishedItem(_ context.Context, _ string, _ *dto.UpdatePublishedItemRequest, _ string) (*dto.ScheduleItemResponse, error) {
	return m.updatePublishedResult, m.updatePublishedErr
}
func (m *mockScheduleService) ListChangeLogs(_ context.Context, _ *dto.ScheduleChangeLogListRequest) ([]dto.ScheduleChangeLogResponse, int64, error) {
	return m.changeLogsResult, m.changeLogsTotal, m.changeLogsErr
}
func (m *mockScheduleService) CheckScope(_ context.Context, _ string) (*dto.ScopeCheckResponse, error) {
	return m.scopeResult, m.scopeErr
}

// ── Mock ExportService ──

type mockExportService struct {
	buf      *bytes.Buffer
	filename string
	err      error
}

func (m *mockExportService) ExportSchedule(_ context.Context, _ string) (*bytes.Buffer, string, error) {
	return m.buf, m.filename, m.err
}

// ═══════════════════════════════════════════════════════════
// Test Helpers
// ═══════════════════════════════════════════════════════════

func setupGin() (*gin.Engine, *gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)
	return r, c, w
}

func setAuth(c *gin.Context) {
	c.Set("user_id", "test-user-id")
	c.Set("role", "admin")
	c.Set("department_id", "test-dept-id")
	c.Set("token_jti", "test-jti")
	c.Set("token_exp", time.Now().Add(15*time.Minute))
}

func jsonBody(v interface{}) io.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

func parseResponse(w *httptest.ResponseRecorder) response.Response {
	var resp response.Response
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp
}

// ═══════════════════════════════════════════════════════════
// AuthHandler Tests
// ═══════════════════════════════════════════════════════════

func TestAuthHandler_Login_Success(t *testing.T) {
	mock := &mockAuthService{
		loginResult: &dto.TokenResponse{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			ExpiresIn:    900,
		},
	}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/auth/login", jsonBody(dto.LoginRequest{
		StudentID: "2024001",
		Password:  "Test1234",
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.POST("/auth/login", h.Login)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	resp := parseResponse(w)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
	// 验证 Set-Cookie 头
	cookies := w.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			found = true
			if c.Value != "test-refresh-token" {
				t.Errorf("expected cookie value test-refresh-token, got %s", c.Value)
			}
		}
	}
	if !found {
		t.Error("expected refresh_token cookie to be set")
	}
}

func TestAuthHandler_Login_BadJSON(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.POST("/auth/login", h.Login)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	mock := &mockAuthService{loginErr: service.ErrInvalidCredentials}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/auth/login", jsonBody(dto.LoginRequest{
		StudentID: "2024001",
		Password:  "wrong",
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.POST("/auth/login", h.Login)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	resp := parseResponse(w)
	if resp.Code != 11001 {
		t.Errorf("expected error code 11001, got %d", resp.Code)
	}
}

func TestAuthHandler_RefreshToken_Success(t *testing.T) {
	mock := &mockAuthService{
		refreshResult: &dto.TokenResponse{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			ExpiresIn:    900,
		},
	}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/auth/refresh", jsonBody(dto.RefreshTokenRequest{
		RefreshToken: "old-refresh",
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.POST("/auth/refresh", h.RefreshToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandler_RefreshToken_MissingToken(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/auth/refresh", jsonBody(map[string]string{}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.POST("/auth/refresh", h.RefreshToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_RefreshToken_FromCookie(t *testing.T) {
	mock := &mockAuthService{
		refreshResult: &dto.TokenResponse{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			ExpiresIn:    900,
		},
	}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "cookie-refresh"})

	r := gin.New()
	r.POST("/auth/refresh", h.RefreshToken)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandler_GetCurrentUser_Success(t *testing.T) {
	mock := &mockAuthService{
		getCurrentResult: &dto.UserDetailResponse{
			ID:   "test-user-id",
			Name: "Test User",
		},
	}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("GET", "/auth/me", nil)

	r := gin.New()
	r.GET("/auth/me", func(c *gin.Context) {
		setAuth(c)
		h.GetCurrentUser(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandler_GetCurrentUser_Unauthenticated(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("GET", "/auth/me", nil)

	r := gin.New()
	r.GET("/auth/me", h.GetCurrentUser)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthHandler_ChangePassword_Success(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("PUT", "/auth/password", jsonBody(dto.ChangePasswordRequest{
		OldPassword: "Old12345",
		NewPassword: "New12345",
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.PUT("/auth/password", func(c *gin.Context) {
		setAuth(c)
		h.ChangePassword(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthHandler_ChangePassword_WeakPassword(t *testing.T) {
	mock := &mockAuthService{changePassErr: service.ErrWeakPassword}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("PUT", "/auth/password", jsonBody(dto.ChangePasswordRequest{
		OldPassword: "Old12345",
		NewPassword: "weak",
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.PUT("/auth/password", func(c *gin.Context) {
		setAuth(c)
		h.ChangePassword(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	mock := &mockAuthService{}
	h := NewAuthHandler(mock, nil)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/auth/logout", nil)

	r := gin.New()
	r.POST("/auth/logout", func(c *gin.Context) {
		setAuth(c)
		h.Logout(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// 验证 Cookie 被清除（max-age = -1）
	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "refresh_token" && c.MaxAge >= 0 {
			t.Error("expected refresh_token cookie to be cleared")
		}
	}
}

// ═══════════════════════════════════════════════════════════
// ScheduleHandler Tests
// ═══════════════════════════════════════════════════════════

func TestScheduleHandler_AutoSchedule_Success(t *testing.T) {
	mock := &mockScheduleService{
		autoResult: &dto.AutoScheduleResponse{
			TotalSlots:  10,
			FilledSlots: 8,
		},
	}
	h := NewScheduleHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/schedules/auto", jsonBody(dto.AutoScheduleRequest{
		SemesterID: "22222222-2222-2222-2222-222222222222",
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.POST("/schedules/auto", func(c *gin.Context) {
		setAuth(c)
		h.AutoSchedule(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestScheduleHandler_AutoSchedule_BadJSON(t *testing.T) {
	mock := &mockScheduleService{}
	h := NewScheduleHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/schedules/auto", bytes.NewReader([]byte("bad")))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.POST("/schedules/auto", func(c *gin.Context) {
		setAuth(c)
		h.AutoSchedule(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestScheduleHandler_AutoSchedule_SubmissionIncomplete(t *testing.T) {
	mock := &mockScheduleService{autoErr: service.ErrSubmissionRateIncomplete}
	h := NewScheduleHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/schedules/auto", jsonBody(dto.AutoScheduleRequest{
		SemesterID: "22222222-2222-2222-2222-222222222222",
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.POST("/schedules/auto", func(c *gin.Context) {
		setAuth(c)
		h.AutoSchedule(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	resp := parseResponse(w)
	if resp.Code != 13107 {
		t.Errorf("expected error code 13107, got %d", resp.Code)
	}
}

func TestScheduleHandler_GetSchedule_MissingSemesterID(t *testing.T) {
	mock := &mockScheduleService{}
	h := NewScheduleHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("GET", "/schedules", nil) // no semester_id

	r := gin.New()
	r.GET("/schedules", h.GetSchedule)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestScheduleHandler_GetSchedule_NotFound(t *testing.T) {
	mock := &mockScheduleService{getErr: service.ErrScheduleNotFound}
	h := NewScheduleHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("GET", "/schedules?semester_id=test", nil)

	r := gin.New()
	r.GET("/schedules", h.GetSchedule)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestScheduleHandler_UpdateItem_Success(t *testing.T) {
	mock := &mockScheduleService{
		updateItemResult: &dto.ScheduleItemResponse{
			ID: "item-1",
		},
	}
	h := NewScheduleHandler(mock)

	memberID := "33333333-3333-3333-3333-333333333333"
	_, _, w := setupGin()
	req := httptest.NewRequest("PUT", "/schedules/items/item-1", jsonBody(dto.UpdateScheduleItemRequest{
		MemberID: &memberID,
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.PUT("/schedules/items/:id", func(c *gin.Context) {
		setAuth(c)
		h.UpdateItem(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestScheduleHandler_UpdateItem_NotDraft(t *testing.T) {
	mock := &mockScheduleService{updateItemErr: service.ErrScheduleNotDraft}
	h := NewScheduleHandler(mock)

	memberID := "33333333-3333-3333-3333-333333333333"
	_, _, w := setupGin()
	req := httptest.NewRequest("PUT", "/schedules/items/item-1", jsonBody(dto.UpdateScheduleItemRequest{
		MemberID: &memberID,
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.PUT("/schedules/items/:id", func(c *gin.Context) {
		setAuth(c)
		h.UpdateItem(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	resp := parseResponse(w)
	if resp.Code != 13104 {
		t.Errorf("expected error code 13104, got %d", resp.Code)
	}
}

func TestScheduleHandler_Publish_Success(t *testing.T) {
	mock := &mockScheduleService{
		publishResult: &dto.ScheduleResponse{
			ID:     "sched-1",
			Status: "published",
		},
	}
	h := NewScheduleHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("POST", "/schedules/publish", jsonBody(dto.PublishScheduleRequest{
		ScheduleID: "44444444-4444-4444-4444-444444444444",
	}))
	req.Header.Set("Content-Type", "application/json")

	r := gin.New()
	r.POST("/schedules/publish", func(c *gin.Context) {
		setAuth(c)
		h.Publish(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestScheduleHandler_CheckScope_Success(t *testing.T) {
	mock := &mockScheduleService{
		scopeResult: &dto.ScopeCheckResponse{
			Changed:    true,
			AddedUsers: []string{"新成员"},
		},
	}
	h := NewScheduleHandler(mock)

	_, _, w := setupGin()
	// CheckScope is now POST (was GET, fixed in this PR)
	req := httptest.NewRequest("POST", "/schedules/sched-1/scope-check", nil)

	r := gin.New()
	r.POST("/schedules/:id/scope-check", func(c *gin.Context) {
		setAuth(c)
		h.CheckScope(c)
	})
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestScheduleHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   int
	}{
		{"NotFound", service.ErrScheduleNotFound, 404, 13101},
		{"ItemNotFound", service.ErrScheduleItemNotFound, 404, 13102},
		{"NotDraft", service.ErrScheduleNotDraft, 400, 13104},
		{"NotPublished", service.ErrScheduleNotPublished, 400, 13105},
		{"CannotPublish", service.ErrScheduleCannotPublish, 400, 13106},
		{"SubmissionRate", service.ErrSubmissionRateIncomplete, 400, 13107},
		{"NoMembers", service.ErrNoEligibleMembers, 400, 13108},
		{"NoTimeSlots", service.ErrNoActiveTimeSlots, 400, 13109},
		{"CandidateNA", service.ErrCandidateNotAvailable, 400, 13110},
		{"SemesterNotFound", service.ErrSemesterNotFound, 404, 13111},
		{"InternalError", errors.New("unknown"), 500, 50000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockScheduleService{getErr: tt.err}
			h := NewScheduleHandler(mock)

			_, _, w := setupGin()
			req := httptest.NewRequest("GET", "/schedules?semester_id=test", nil)

			r := gin.New()
			r.GET("/schedules", h.GetSchedule)
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, w.Code)
			}
			resp := parseResponse(w)
			if resp.Code != tt.wantCode {
				t.Errorf("expected code %d, got %d", tt.wantCode, resp.Code)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════
// ExportHandler Tests
// ═══════════════════════════════════════════════════════════

func TestExportHandler_Success(t *testing.T) {
	buf := bytes.NewBufferString("excel content")
	mock := &mockExportService{
		buf:      buf,
		filename: "排班表_2025秋.xlsx",
	}
	h := NewExportHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("GET", "/export/schedule?semester_id=test", nil)

	r := gin.New()
	r.GET("/export/schedule", h.ExportSchedule)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("unexpected content type: %s", ct)
	}
	cd := w.Header().Get("Content-Disposition")
	if cd == "" {
		t.Error("expected Content-Disposition header")
	}
}

func TestExportHandler_MissingSemesterID(t *testing.T) {
	mock := &mockExportService{}
	h := NewExportHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("GET", "/export/schedule", nil)

	r := gin.New()
	r.GET("/export/schedule", h.ExportSchedule)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestExportHandler_NoSchedule(t *testing.T) {
	mock := &mockExportService{err: service.ErrExportNoSchedule}
	h := NewExportHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("GET", "/export/schedule?semester_id=test", nil)

	r := gin.New()
	r.GET("/export/schedule", h.ExportSchedule)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestExportHandler_NoItems(t *testing.T) {
	mock := &mockExportService{err: service.ErrExportNoItems}
	h := NewExportHandler(mock)

	_, _, w := setupGin()
	req := httptest.NewRequest("GET", "/export/schedule?semester_id=test", nil)

	r := gin.New()
	r.GET("/export/schedule", h.ExportSchedule)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
