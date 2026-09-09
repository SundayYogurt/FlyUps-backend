package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flyup/config"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/oauth2"
)

const testSecret = "test-secret-key-for-testing-1234"

// ─── Mock: UserService ─────────────────────────────────────────────────────────

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) SignUp(ctx context.Context, input dto.UserSignUp) (string, error) {
	args := m.Called(ctx, input)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) Signing(ctx context.Context, email, password string) (string, uint, string, error) {
	args := m.Called(ctx, email, password)
	return args.String(0), args.Get(1).(uint), args.String(2), args.Error(3)
}

func (m *MockUserService) GoogleSigning(ctx context.Context, code, role string, cfg *oauth2.Config) (string, error) {
	args := m.Called(ctx, code, role, cfg)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) VerifyEmail(ctx context.Context, input dto.VerifyEmailRequest) (string, error) {
	args := m.Called(ctx, input)
	return args.String(0), args.Error(1)
}

func (m *MockUserService) ForgotPassword(ctx context.Context, email string) error {
	return m.Called(ctx, email).Error(0)
}

func (m *MockUserService) SetPassword(token, newPassword string) error {
	return m.Called(token, newPassword).Error(0)
}

func (m *MockUserService) ChangePassword(userID uint, old, newPwd string) error {
	return m.Called(userID, old, newPwd).Error(0)
}

func (m *MockUserService) AddPasswordForGoogle(userID uint, newPassword string) error {
	return m.Called(userID, newPassword).Error(0)
}

func (m *MockUserService) GetProfile(ctx context.Context, userID uint) (*domain.User, error) {
	args := m.Called(ctx, userID)
	if v := args.Get(0); v != nil {
		return v.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) UpdateProfile(ctx context.Context, userID uint, input dto.ProfileInput) error {
	return m.Called(ctx, userID, input).Error(0)
}

func (m *MockUserService) VerifyStudent(userID uint, input dto.VerifyStudentInput) error {
	return m.Called(userID, input).Error(0)
}

func (m *MockUserService) VerifyID(userID uint, input dto.VerifyIDInput) error {
	return m.Called(userID, input).Error(0)
}

func (m *MockUserService) GetAllStudentVerifyRequest() ([]domain.StudentCardVerification, error) {
	args := m.Called()
	return args.Get(0).([]domain.StudentCardVerification), args.Error(1)
}

func (m *MockUserService) GetAllCardIDVerifyRequest() ([]domain.IdCardVerification, error) {
	args := m.Called()
	return args.Get(0).([]domain.IdCardVerification), args.Error(1)
}

func (m *MockUserService) AddBankAccount(userID uint, input dto.BankRequest) error {
	return m.Called(userID, input).Error(0)
}

func (m *MockUserService) UpdateBankAccount(userID, bankID uint, input dto.BankRequest) error {
	return m.Called(userID, bankID, input).Error(0)
}

func (m *MockUserService) FindBankByUserID(id uint) ([]domain.BankAccount, error) {
	args := m.Called(id)
	return args.Get(0).([]domain.BankAccount), args.Error(1)
}

func (m *MockUserService) SetDefaultBankAccount(userID, bankID uint) error {
	return m.Called(userID, bankID).Error(0)
}

func (m *MockUserService) ApproveIdCard(userID, adminID uint) error {
	return m.Called(userID, adminID).Error(0)
}

func (m *MockUserService) ApproveStudentCard(userID, adminID uint) error {
	return m.Called(userID, adminID).Error(0)
}

func (m *MockUserService) RejectIdCard(userID, adminID uint) error {
	return m.Called(userID, adminID).Error(0)
}

func (m *MockUserService) RejectStudentCard(userID, adminID uint) error {
	return m.Called(userID, adminID).Error(0)
}

func (m *MockUserService) SuspendUser(adminID, userID uint, reason string) error {
	return m.Called(adminID, userID, reason).Error(0)
}

func (m *MockUserService) RollbackActiveUser(userID uint) error {
	return m.Called(userID).Error(0)
}

func (m *MockUserService) GetNotificationPreferences(userID uint) (map[string]bool, error) {
	args := m.Called(userID)
	return args.Get(0).(map[string]bool), args.Error(1)
}

func (m *MockUserService) UpdateNotificationPreferences(userID uint, prefs map[string]bool) error {
	return m.Called(userID, prefs).Error(0)
}

func (m *MockUserService) ListUser(page, limit int, role, status, search string) ([]domain.User, int64, error) {
	args := m.Called(page, limit, role, status, search)
	return args.Get(0).([]domain.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserService) CreateUniversity(req dto.CreateUniversityRequest) (*domain.University, error) {
	args := m.Called(req)
	if v := args.Get(0); v != nil {
		return v.(*domain.University), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) GetAllUniversities() ([]domain.University, error) {
	args := m.Called()
	return args.Get(0).([]domain.University), args.Error(1)
}

func (m *MockUserService) GetUniversityByID(id uint) (*domain.University, error) {
	args := m.Called(id)
	if v := args.Get(0); v != nil {
		return v.(*domain.University), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) UpdateUniversity(id uint, req dto.CreateUniversityRequest) (*domain.University, error) {
	args := m.Called(id, req)
	if v := args.Get(0); v != nil {
		return v.(*domain.University), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) DeleteUniversity(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockUserService) CreateDomain(ctx context.Context, id uint, req dto.CreateDomainRequest) (*domain.UniversityDomain, error) {
	args := m.Called(ctx, id, req)
	if v := args.Get(0); v != nil {
		return v.(*domain.UniversityDomain), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) GetUniversityByEmail(ctx context.Context, email string) (*domain.UniversityDomain, error) {
	args := m.Called(ctx, email)
	if v := args.Get(0); v != nil {
		return v.(*domain.UniversityDomain), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) DeleteDomain(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockUserService) UpdateDomain(ctx context.Context, id uint, req dto.UpdateDomainRequest) (*domain.UniversityDomain, error) {
	args := m.Called(ctx, id, req)
	if v := args.Get(0); v != nil {
		return v.(*domain.UniversityDomain), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserService) SelectRole(ctx context.Context, userID uint, newRole string) error {
	return m.Called(ctx, userID, newRole).Error(0)
}

// ─── Mock: AdminLogService ─────────────────────────────────────────────────────

type MockAdminLogService struct{}

func (m *MockAdminLogService) LogAction(_ uint, _, _ string, _ *uint, _ *string) {}
func (m *MockAdminLogService) ListLogs(_ dto.AdminLogFilter) ([]dto.AdminLogItem, int64, error) {
	return nil, 0, nil
}

// ─── Setup ─────────────────────────────────────────────────────────────────────

func setupTest(t *testing.T) (*fiber.App, *MockUserService, *UserHandler) {
	app := fiber.New()
	mockService := new(MockUserService)
	handler := NewUserHandler(
		mockService,
		helper.Auth{Secret: testSecret},
		&oauth2.Config{},
		config.AppConfig{AppSecret: testSecret},
		&MockAdminLogService{},
	)
	return app, mockService, handler
}

// ─── SignUp ────────────────────────────────────────────────────────────────────

func TestUserHandler_SignUp(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Post("/signup", handler.SignUp)

	body := dto.UserSignUp{Role: "booster", FirstName: "user", LastName: "test", Email: "test@test.com", Phone: "0999999999", Password: "T@st12345", AcceptTerms: true}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("SignUp", mock.Anything, mock.Anything).Return("signup success", nil)
	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
}

func TestUserHandler_SignUp_ValidationFail(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Post("/signup", handler.SignUp)

	body := dto.UserSignUp{Email: "not-an-email"} // ขาด field required
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUserHandler_SignUp_DuplicateEmail(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Post("/signup", handler.SignUp)

	body := dto.UserSignUp{Role: "booster", FirstName: "user", LastName: "test", Email: "test@test.com", Phone: "0999999999", Password: "T@st12345", AcceptTerms: true}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("SignUp", mock.Anything, mock.Anything).Return("", errors.New("already registered"))
	req := httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

// ─── VerifyEmail ───────────────────────────────────────────────────────────────

func TestUserHandler_VerifyEmail(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Get("/verify-email", handler.VerifyEmail)

	mockService.On("VerifyEmail", mock.Anything, mock.Anything).Return("email verified", nil)
	req := httptest.NewRequest(http.MethodGet, "/verify-email?token=valid-token", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_VerifyEmail_MissingToken(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Get("/verify-email", handler.VerifyEmail)

	req := httptest.NewRequest(http.MethodGet, "/verify-email", nil) // ไม่ส่ง token
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Signing ───────────────────────────────────────────────────────────────────

func TestUserHandler_Signing(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Post("/signin", handler.Signing)

	body := dto.UserSigning{Email: "test@test.com", Password: "password"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("Signing", mock.Anything, "test@test.com", "password").Return("access-token", uint(1), "booster", nil)
	req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_Signing_UnverifiedEmail(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Post("/signin", handler.Signing)

	body := dto.UserSigning{Email: "test@test.com", Password: "password"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("Signing", mock.Anything, mock.Anything, mock.Anything).Return("", uint(0), "", errors.New("please verify your email first"))
	req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUserHandler_Signing_InvalidCredentials(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Post("/signin", handler.Signing)

	body := dto.UserSigning{Email: "wrong@test.com", Password: "wrongpass"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("Signing", mock.Anything, mock.Anything, mock.Anything).Return("", uint(0), "", errors.New("wrong password"))
	req := httptest.NewRequest(http.MethodPost, "/signin", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── ForgotPassword ────────────────────────────────────────────────────────────

func TestUserHandler_ForgotPassword(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Post("/forgot-password", handler.ForgotPassword)

	body := dto.ForgotPasswordRequest{Email: "test@test.com"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("ForgotPassword", mock.Anything, "test@test.com").Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_ForgotPassword_NotFound(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Post("/forgot-password", handler.ForgotPassword)

	body := dto.ForgotPasswordRequest{Email: "none@test.com"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("ForgotPassword", mock.Anything, mock.Anything).Return(errors.New("user not found"))
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── SetPassword ───────────────────────────────────────────────────────────────

func TestUserHandler_SetPassword(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Post("/reset-password", handler.SetPassword)

	body := map[string]string{"new_password": "NewPass@123"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("SetPassword", "valid-token", "NewPass@123").Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/reset-password?reset_token=valid-token", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_SetPassword_MissingToken(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Post("/reset-password", handler.SetPassword)

	body := map[string]string{"new_password": "NewPass@123"}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/reset-password", bytes.NewBuffer(bodyJSON)) // ไม่มี reset_token
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Me ────────────────────────────────────────────────────────────────────────

func TestUserHandler_Me(t *testing.T) {
	app, mockService, handler := setupTest(t)
	// inject user เข้า ctx.Locals ก่อน route จะทำงาน
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/user/me", handler.Me)

	mockService.On("GetProfile", mock.Anything, uint(1)).Return(&domain.User{ID: 1, Email: "test@test.com"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/user/me", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_Me_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Get("/user/me", handler.Me) // ไม่มี middleware inject user

	req := httptest.NewRequest(http.MethodGet, "/user/me", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── UpdateProfile ─────────────────────────────────────────────────────────────

func TestUserHandler_UpdateProfile(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Patch("/user/profile", handler.UpdateProfile)

	firstName := "Updated"
	body := dto.ProfileInput{FirstName: &firstName}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("UpdateProfile", mock.Anything, uint(1), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/user/profile", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── SignOut ───────────────────────────────────────────────────────────────────

func TestUserHandler_SignOut(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Post("/user/signout", handler.SignOut)

	req := httptest.NewRequest(http.MethodPost, "/user/signout", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── SelectRole ────────────────────────────────────────────────────────────────

func TestUserHandler_SelectRole(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pending"})
		return c.Next()
	})
	app.Patch("/user/role", handler.SelectRole)

	body := map[string]string{"role": "booster"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("SelectRole", mock.Anything, uint(1), "booster").Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/user/role", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── AddBankAccount ────────────────────────────────────────────────────────────

func TestUserHandler_AddBankAccount(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/user/add-bank", handler.AddBankAccount)

	bankName := "SCB"
	body := dto.BankRequest{BankName: &bankName}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("AddBankAccount", uint(1), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/user/add-bank", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── UpdateBankAccount ─────────────────────────────────────────────────────────

func TestUserHandler_UpdateBankAccount(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/user/update-bank/:id", handler.UpdateBankAccount)

	bankName := "KBank"
	body := dto.BankRequest{BankName: &bankName}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("UpdateBankAccount", uint(1), uint(5), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/user/update-bank/5", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_UpdateBankAccount_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/user/update-bank/:id", handler.UpdateBankAccount)

	body := map[string]string{}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/user/update-bank/abc", bytes.NewBuffer(bodyJSON)) // id ไม่ใช่ตัวเลข
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── ChangePassword ────────────────────────────────────────────────────────────

func TestUserHandler_ChangePassword(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Put("/user/change-password", handler.ChangePassword)

	body := dto.ChangePasswordRequest{OldPassword: "OldPass@1", NewPassword: "NewPass@1"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("ChangePassword", uint(1), "OldPass@1", "NewPass@1").Return(nil)
	req := httptest.NewRequest(http.MethodPut, "/user/change-password", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── RefreshToken ──────────────────────────────────────────────────────────────

func TestUserHandler_RefreshToken_Valid(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Post("/auth/refresh", handler.RefreshToken)

	// สร้าง refresh token จริงด้วย secret เดียวกัน
	auth := helper.Auth{Secret: testSecret}
	token, _ := auth.GenerateRefreshToken(1, "test@test.com", "booster")

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.Header.Set("X-Refresh-Token", token)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_RefreshToken_Missing(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Post("/auth/refresh", handler.RefreshToken)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil) // ไม่ส่ง token
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── GetNotificationPreferences ───────────────────────────────────────────────

func TestUserHandler_GetNotificationPreferences(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Get("/user/notification-preferences", handler.GetNotificationPreferences)

	mockService.On("GetNotificationPreferences", uint(1)).Return(map[string]bool{"email": true}, nil)
	req := httptest.NewRequest(http.MethodGet, "/user/notification-preferences", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── Admin: GetBankUserAccounts ────────────────────────────────────────────────

func TestUserHandler_GetBankUserAccounts(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Get("/admin/user-banks/:id", handler.GetBankUserAccounts)

	mockService.On("FindBankByUserID", uint(3)).Return([]domain.BankAccount{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/user-banks/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_GetBankUserAccounts_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Get("/admin/user-banks/:id", handler.GetBankUserAccounts)

	req := httptest.NewRequest(http.MethodGet, "/admin/user-banks/abc", nil) // id ไม่ใช่ตัวเลข
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Admin: ApproveStudentCard ─────────────────────────────────────────────────

func TestUserHandler_ApproveStudentCard(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/approve-student-card/:id", handler.ApproveStudentCard)

	mockService.On("ApproveStudentCard", uint(7), uint(99)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/approve-student-card/7", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── Admin: RejectStudentCard ──────────────────────────────────────────────────

func TestUserHandler_RejectStudentCard(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/reject-student-card/:id", handler.RejectStudentCard)

	mockService.On("RejectStudentCard", uint(7), uint(99)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/reject-student-card/7", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── Admin: SuspendUser ────────────────────────────────────────────────────────

func TestUserHandler_SuspendUser(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/suspend-user/:id", handler.SuspendUser)

	body := dto.SuspendUserInput{Reason: "violated rules"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("SuspendUser", uint(99), uint(5), "violated rules").Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/suspend-user/5", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_SuspendUser_MissingReason(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/suspend-user/:id", handler.SuspendUser)

	body := dto.SuspendUserInput{Reason: ""} // reason ว่าง
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/admin/suspend-user/5", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── Admin: RollbackUser ───────────────────────────────────────────────────────

func TestUserHandler_RollbackUser(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/rollback-user/:id", handler.RollbackUser)

	mockService.On("RollbackActiveUser", uint(5)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/rollback-user/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── Admin: ListUsers ──────────────────────────────────────────────────────────

func TestUserHandler_ListUsers(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/list-users", handler.ListUsers)

	mockService.On("ListUser", 1, 10, "", "", "").Return([]domain.User{}, int64(0), nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/list-users", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── Admin: GetStudentsCardRequest ─────────────────────────────────────────────

func TestUserHandler_GetStudentsCardRequest(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/student-verifications", handler.GetStudentsCardRequest)

	mockService.On("GetAllStudentVerifyRequest").Return([]domain.StudentCardVerification{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/student-verifications", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── Admin: CreateUniversity ───────────────────────────────────────────────────

func TestUserHandler_CreateUniversity(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Post("/admin/create-university", handler.CreateUniversity)

	nameEN := "Test University"
	body := dto.CreateUniversityRequest{NameEN: &nameEN}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("CreateUniversity", mock.Anything).Return(&domain.University{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/create-university", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── VerifyStudent ─────────────────────────────────────────────────────────────

func TestUserHandler_VerifyStudent(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/user/student-verify", handler.VerifyStudent)

	cardURL := "https://example.com/card.jpg"
	body := dto.VerifyStudentInput{StudentCardURL: &cardURL}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("VerifyStudent", uint(1), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/user/student-verify", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_VerifyStudent_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Post("/user/student-verify", handler.VerifyStudent)

	req := httptest.NewRequest(http.MethodPost, "/user/student-verify", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── VerifyIDCard ──────────────────────────────────────────────────────────────

func TestUserHandler_VerifyIDCard(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Post("/user/id-verify", handler.VerifyIDCard)

	idURL := "https://example.com/id.jpg"
	selfieURL := "https://example.com/selfie.jpg"
	body := dto.VerifyIDInput{IDCardURL: &idURL, SelfieURL: &selfieURL}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("VerifyID", uint(1), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPost, "/user/id-verify", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_VerifyIDCard_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Post("/user/id-verify", handler.VerifyIDCard)

	req := httptest.NewRequest(http.MethodPost, "/user/id-verify", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── SetDefaultBankAccount ─────────────────────────────────────────────────────

func TestUserHandler_SetDefaultBankAccount(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/user/set-default-bank/:id", handler.SetDefaultBankAccount)

	mockService.On("SetDefaultBankAccount", uint(1), uint(5)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/user/set-default-bank/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_SetDefaultBankAccount_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Patch("/user/set-default-bank/:id", handler.SetDefaultBankAccount)

	req := httptest.NewRequest(http.MethodPatch, "/user/set-default-bank/5", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_SetDefaultBankAccount_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "pioneer"})
		return c.Next()
	})
	app.Patch("/user/set-default-bank/:id", handler.SetDefaultBankAccount)

	req := httptest.NewRequest(http.MethodPatch, "/user/set-default-bank/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── ApproveCardID ─────────────────────────────────────────────────────────────

func TestUserHandler_ApproveCardID(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/approve-id-card/:id", handler.ApproveCardID)

	mockService.On("ApproveIdCard", uint(7), uint(99)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/approve-id-card/7", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_ApproveCardID_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Patch("/admin/approve-id-card/:id", handler.ApproveCardID)

	req := httptest.NewRequest(http.MethodPatch, "/admin/approve-id-card/7", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_ApproveCardID_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/approve-id-card/:id", handler.ApproveCardID)

	req := httptest.NewRequest(http.MethodPatch, "/admin/approve-id-card/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── RejectCardID ──────────────────────────────────────────────────────────────

func TestUserHandler_RejectCardID(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/reject-id-card/:id", handler.RejectCardID)

	mockService.On("RejectIdCard", uint(7), uint(99)).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/admin/reject-id-card/7", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_RejectCardID_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Patch("/admin/reject-id-card/:id", handler.RejectCardID)

	req := httptest.NewRequest(http.MethodPatch, "/admin/reject-id-card/7", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_RejectCardID_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Patch("/admin/reject-id-card/:id", handler.RejectCardID)

	req := httptest.NewRequest(http.MethodPatch, "/admin/reject-id-card/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── GetCardIDRequests ─────────────────────────────────────────────────────────

func TestUserHandler_GetCardIDRequests(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/id-card-verifications", handler.GetCardIDRequests)

	mockService.On("GetAllCardIDVerifyRequest").Return([]domain.IdCardVerification{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/id-card-verifications", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// ─── UpdateNotificationPreferences ────────────────────────────────────────────

func TestUserHandler_UpdateNotificationPreferences(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Patch("/user/notification-preferences", handler.UpdateNotificationPreferences)

	body := dto.UpdateNotificationPrefsRequest{Preferences: map[string]bool{"email": true}}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("UpdateNotificationPreferences", uint(1), mock.Anything).Return(nil)
	req := httptest.NewRequest(http.MethodPatch, "/user/notification-preferences", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_UpdateNotificationPreferences_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Patch("/user/notification-preferences", handler.UpdateNotificationPreferences)

	req := httptest.NewRequest(http.MethodPatch, "/user/notification-preferences", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── AddPassword ───────────────────────────────────────────────────────────────

func TestUserHandler_AddPassword(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 1, Email: "test@test.com", Role: "booster"})
		return c.Next()
	})
	app.Put("/user/add-password", handler.AddPassword)

	body := dto.AddPasswordRequest{NewPassword: "NewPass@123"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("AddPasswordForGoogle", uint(1), "NewPass@123").Return(nil)
	req := httptest.NewRequest(http.MethodPut, "/user/add-password", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_AddPassword_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Put("/user/add-password", handler.AddPassword)

	req := httptest.NewRequest(http.MethodPut, "/user/add-password", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── GetUniversity ─────────────────────────────────────────────────────────────

func TestUserHandler_GetUniversity(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/university/:id", handler.GetUniversity)

	mockService.On("GetUniversityByID", uint(3)).Return(&domain.University{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/university/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_GetUniversity_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Get("/admin/university/:id", handler.GetUniversity)

	req := httptest.NewRequest(http.MethodGet, "/admin/university/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_GetUniversity_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/university/:id", handler.GetUniversity)

	req := httptest.NewRequest(http.MethodGet, "/admin/university/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── GetUniversities ───────────────────────────────────────────────────────────

func TestUserHandler_GetUniversities(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Get("/admin/universities", handler.GetUniversities)

	mockService.On("GetAllUniversities").Return([]domain.University{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/admin/universities", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_GetUniversities_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Get("/admin/universities", handler.GetUniversities)

	req := httptest.NewRequest(http.MethodGet, "/admin/universities", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ─── UpdateUniversity ──────────────────────────────────────────────────────────

func TestUserHandler_UpdateUniversity(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Put("/admin/update-university/:id", handler.UpdateUniversity)

	nameEN := "Updated University"
	body := dto.CreateUniversityRequest{NameEN: &nameEN}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("UpdateUniversity", uint(3), mock.Anything).Return(&domain.University{}, nil)
	req := httptest.NewRequest(http.MethodPut, "/admin/update-university/3", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_UpdateUniversity_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Put("/admin/update-university/:id", handler.UpdateUniversity)

	req := httptest.NewRequest(http.MethodPut, "/admin/update-university/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_UpdateUniversity_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Put("/admin/update-university/:id", handler.UpdateUniversity)

	req := httptest.NewRequest(http.MethodPut, "/admin/update-university/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── DeleteUniversity ──────────────────────────────────────────────────────────

func TestUserHandler_DeleteUniversity(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Delete("/admin/delete-university/:id", handler.DeleteUniversity)

	mockService.On("DeleteUniversity", uint(3)).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/admin/delete-university/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_DeleteUniversity_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Delete("/admin/delete-university/:id", handler.DeleteUniversity)

	req := httptest.NewRequest(http.MethodDelete, "/admin/delete-university/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_DeleteUniversity_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Delete("/admin/delete-university/:id", handler.DeleteUniversity)

	req := httptest.NewRequest(http.MethodDelete, "/admin/delete-university/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── CreateUniversityDomain ────────────────────────────────────────────────────

func TestUserHandler_CreateUniversityDomain(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Post("/admin/create-university-domain/:id", handler.CreateUniversityDomain)

	body := dto.CreateDomainRequest{Domain: "example.ac.th"}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("CreateDomain", mock.Anything, uint(3), mock.Anything).Return(&domain.UniversityDomain{}, nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/create-university-domain/3", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_CreateUniversityDomain_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Post("/admin/create-university-domain/:id", handler.CreateUniversityDomain)

	req := httptest.NewRequest(http.MethodPost, "/admin/create-university-domain/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_CreateUniversityDomain_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Post("/admin/create-university-domain/:id", handler.CreateUniversityDomain)

	req := httptest.NewRequest(http.MethodPost, "/admin/create-university-domain/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── UpdateUniversityDomain ────────────────────────────────────────────────────

func TestUserHandler_UpdateUniversityDomain(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Put("/admin/update-university-domain/:id", handler.UpdateUniversityDomain)

	newDomain := "new.ac.th"
	body := dto.UpdateDomainRequest{Domain: &newDomain}
	bodyJSON, _ := json.Marshal(body)
	mockService.On("UpdateDomain", mock.Anything, uint(3), mock.Anything).Return(&domain.UniversityDomain{}, nil)
	req := httptest.NewRequest(http.MethodPut, "/admin/update-university-domain/3", bytes.NewBuffer(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_UpdateUniversityDomain_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Put("/admin/update-university-domain/:id", handler.UpdateUniversityDomain)

	req := httptest.NewRequest(http.MethodPut, "/admin/update-university-domain/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_UpdateUniversityDomain_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Put("/admin/update-university-domain/:id", handler.UpdateUniversityDomain)

	req := httptest.NewRequest(http.MethodPut, "/admin/update-university-domain/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// ─── DeleteUniversityDomain ────────────────────────────────────────────────────

func TestUserHandler_DeleteUniversityDomain(t *testing.T) {
	app, mockService, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Delete("/admin/delete-university-domain/:id", handler.DeleteUniversityDomain)

	mockService.On("DeleteDomain", uint(3)).Return(nil)
	req := httptest.NewRequest(http.MethodDelete, "/admin/delete-university-domain/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestUserHandler_DeleteUniversityDomain_Unauthorized(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Delete("/admin/delete-university-domain/:id", handler.DeleteUniversityDomain)

	req := httptest.NewRequest(http.MethodDelete, "/admin/delete-university-domain/3", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUserHandler_DeleteUniversityDomain_InvalidID(t *testing.T) {
	app, _, handler := setupTest(t)
	app.Use(func(c fiber.Ctx) error {
		c.Locals("user", domain.User{ID: 99, Email: "admin@test.com", Role: "admin"})
		return c.Next()
	})
	app.Delete("/admin/delete-university-domain/:id", handler.DeleteUniversityDomain)

	req := httptest.NewRequest(http.MethodDelete, "/admin/delete-university-domain/abc", nil)
	resp, _ := app.Test(req)
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status=%d body=%s", resp.StatusCode, string(respBody))
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
