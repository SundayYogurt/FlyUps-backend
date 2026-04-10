package service

import (
	"flyup/config"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"golang.org/x/oauth2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func ptr[T any](v T) *T {
	return &v
}

type mockUserRepository struct {
	mock.Mock
}

func (m *mockUserRepository) CreateUser(usr *domain.User, consent *domain.UserConsent) (*domain.User, error) {
	args := m.Called(usr, consent)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) UpdateUserProfile(userID uint, firstName, lastName, phone string, address *string) error {
	args := m.Called(userID, firstName, lastName, phone, address)
	return args.Error(0)
}

func (m *mockUserRepository) UpsertStudentProfileByUserID(profile *domain.StudentProfile) error {
	args := m.Called(profile)
	return args.Error(0)
}

func (m *mockUserRepository) CreateBankAccount(bank *domain.BankAccount) error {
	args := m.Called(bank)
	return args.Error(0)
}

func (m *mockUserRepository) UpdateBankAccount(bank *domain.BankAccount) error {
	args := m.Called(bank)
	return args.Error(0)
}

func (m *mockUserRepository) FindBankByUserId(userID uint) ([]domain.BankAccount, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).([]domain.BankAccount), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) FindBankById(id uint) (*domain.BankAccount, error) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.BankAccount), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) FindBankByAccountNumber(accountNumber string) (*domain.BankAccount, error) {
	args := m.Called(accountNumber)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.BankAccount), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) UpdateIdCardVerification(v *domain.IdCardVerification) error {
	args := m.Called(v)
	return args.Error(0)
}

func (m *mockUserRepository) UpdateStudentCardVerification(v *domain.StudentCardVerification) error {
	args := m.Called(v)
	return args.Error(0)
}

func (m *mockUserRepository) FindStudentStatus(userID uint) (*domain.StudentCardVerification, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.StudentCardVerification), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) FindIdCardStatus(userID uint) (*domain.IdCardVerification, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.IdCardVerification), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) CreateIdVerification(v *domain.IdCardVerification) error {
	args := m.Called(v)
	return args.Error(0)
}

func (m *mockUserRepository) CreateConsents(consents []*domain.UserConsent) error {
	args := m.Called(consents)
	return args.Error(0)
}

func (m *mockUserRepository) UpdateIdVerification(v *domain.IdCardVerification) error {
	args := m.Called(v)
	return args.Error(0)
}

func (m *mockUserRepository) FindLatestIdVerification(userID uint) (*domain.IdCardVerification, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.IdCardVerification), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) UpdateStudentVerification(v *domain.StudentCardVerification) error {
	args := m.Called(v)
	return args.Error(0)
}

func (m *mockUserRepository) FindLatestStudentVerification(userID uint) (*domain.StudentCardVerification, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.StudentCardVerification), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) CreateStudentVerification(v *domain.StudentCardVerification) error {
	args := m.Called(v)
	return args.Error(0)
}

func (m *mockUserRepository) UpdateUser(userID uint, updates map[string]interface{}) error {
	args := m.Called(userID, updates)
	return args.Error(0)
}

func (m *mockUserRepository) FindUser(email string) (*domain.User, error) {
	args := m.Called(email)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepository) FindUserByVerificationToken(token string) (*domain.User, error) {
	args := m.Called(token)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepository) FindUserByResetToken(token string) (*domain.User, error) {
	args := m.Called(token)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepository) FindUserById(id uint) (*domain.User, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.User), args.Error(1)
}

type mockAuth struct {
	mock.Mock
}

func (m *mockAuth) VerifyPassword(password string, hash string) error {
	args := m.Called(password, hash)
	return args.Error(0)
}

func (m *mockAuth) GenerateToken(userID uint, email string, role string) (string, error) {
	args := m.Called(userID, email, role)
	return args.String(0), args.Error(1)
}

func (m *mockAuth) VerifyToken(token string) (domain.User, error) {
	args := m.Called(token)

	user, _ := args.Get(0).(domain.User)
	return user, args.Error(1)
}

func (m *mockAuth) CreateHashedPassword(pw string) (string, error) {
	args := m.Called(pw)
	return args.String(0), args.Error(1)
}

func (m *mockAuth) GenerateCode() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

type mockUniversityRepo struct {
	mock.Mock
}

func TestSignup_Success(t *testing.T) {

	repo := new(mockUserRepository)
	auth := new(mockAuth)

	svc := NewUserService(
		repo,
		nil, // ไม่ใช้ pioneer เลยไม่ต้อง mock
		auth,
		config.AppConfig{
			BaseURL: "http://localhost",
		},
	)

	input := dto.UserSignUp{
		Role:        "booster",
		FirstName:   "John",
		LastName:    "Doe",
		Email:       "test@test.com",
		Phone:       "0812345678",
		Password:    "password123",
		AcceptTerms: true,
	}

	// mock behavior

	repo.On("FindUser", "test@test.com").
		Return(&domain.User{}, gorm.ErrRecordNotFound)

	auth.On("CreateHashedPassword", "password123").
		Return("hashed-password", nil)

	auth.On("GenerateCode").
		Return("verify-token", nil)

	repo.On("CreateUser",
		mock.AnythingOfType("*domain.User"),
		mock.AnythingOfType("*domain.UserConsent"),
	).Return(&domain.User{ID: 1}, nil)

	// run
	msg, err := svc.SignUp(input)

	// assert
	assert.NoError(t, err)
	assert.NotEmpty(t, msg)

	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}

func TestGoogleSignin_NewUser_Success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Mock Google Token Request
	httpmock.RegisterResponder("POST", "https://oauth2.googleapis.com/token",
		httpmock.NewStringResponder(200, `{"access_token": "mock_token", "token_type": "Bearer"}`))

	// Mock Google User Info Endpoint
	httpmock.RegisterResponder("GET", "https://www.googleapis.com/oauth2/v2/userinfo",
		httpmock.NewStringResponder(200, `{"id":"123","email":"test@google.com","given_name":"John","family_name":"Doe"}`))

	repo := new(mockUserRepository)
	auth := new(mockAuth)

	svc := NewUserService(repo, nil, auth, config.AppConfig{})
	oauthConf := &oauth2.Config{
		ClientID: "test", ClientSecret: "test", Endpoint: oauth2.Endpoint{
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	// Define expected DB and Auth actions inside the Service flow
	repo.On("FindUser", "test@google.com").Return(&domain.User{}, gorm.ErrRecordNotFound)
	repo.On("CreateUser", mock.AnythingOfType("*domain.User"), mock.AnythingOfType("*domain.UserConsent")).Return(&domain.User{ID: 1, Email: "test@google.com", Role: "booster"}, nil)
	auth.On("GenerateToken", uint(1), "test@google.com", "booster").Return("mock.jwt.token", nil)

	// Call the method to test
	token, err := svc.GoogleSigning("mock-code", "booster", oauthConf)

	// Assertions
	assert.NoError(t, err)
	assert.Equal(t, "mock.jwt.token", token)

	repo.AssertExpectations(t)
	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}

func TestSigning_Success(t *testing.T) {
	repo := new(mockUserRepository)
	auth := new(mockAuth)
	svc := NewUserService(repo, nil, auth, config.AppConfig{})

	existingUser := &domain.User{
		ID:       2,
		Email:    "test@test.com",
		PasswordHash: "hashed_password",
		Role:            "pioneer",
		Status:          domain.ACTIVE,
		EmailVerifiedAt: ptr(time.Now()),
	}

	repo.On("FindUser", "test@test.com").Return(existingUser, nil)
	auth.On("VerifyPassword", "password123", "hashed_password").Return(nil)
	auth.On("GenerateToken", uint(2), "test@test.com", "pioneer").Return("mock.jwt.token", nil)

	token, err := svc.Signing("test@test.com", "password123")

	assert.NoError(t, err)
	assert.Equal(t, "mock.jwt.token", token)

	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}

func TestVerifyEmail_Success(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{})

	existingUser := &domain.User{
		ID:               3,
		VerificationToken: ptr("123456"),
		VerificationTokenExpiresAt: ptr(time.Now().Add(1 * time.Hour)),
		Status:            domain.SUSPENDED,
	}

	repo.On("FindUserByVerificationToken", "123456").Return(existingUser, nil)
	// After verify, user becomes active and code is cleared
	repo.On("UpdateUser", uint(3), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	msg, err := svc.VerifyEmail(dto.VerifyEmailRequest{Token: "123456"})

	assert.NoError(t, err)
	assert.NotEmpty(t, msg)

	repo.AssertExpectations(t)
}

func TestForgotPassword_Success(t *testing.T) {
	repo := new(mockUserRepository)
	auth := new(mockAuth)
	svc := NewUserService(repo, nil, auth, config.AppConfig{})

	existingUser := &domain.User{
		ID:    4,
		Email:  "forgot@test.com",
		Status: domain.ACTIVE,
	}

	repo.On("FindUser", "forgot@test.com").Return(existingUser, nil)
	repo.On("UpdateUser", uint(4), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", "https://api.resend.com/emails", httpmock.NewStringResponder(200, `{}`))

	err := svc.ForgotPassword("forgot@test.com")

	assert.NoError(t, err)

	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}

func TestSetPassword_Success(t *testing.T) {
	repo := new(mockUserRepository)
	auth := new(mockAuth)
	svc := NewUserService(repo, nil, auth, config.AppConfig{})

	existingUser := &domain.User{
		ID:                  5,
		ResetTokenHash:      ptr("valid-token"),
		ResetTokenExpiresAt: ptr(time.Now().Add(1 * time.Hour)), // Valid expiry
	}

	repo.On("FindUserByResetToken", mock.AnythingOfType("string")).Return(existingUser, nil)
	repo.On("UpdateUser", uint(5), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	err := svc.SetPassword("valid-token", "NewPassword123!")

	assert.NoError(t, err)

	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}
