package services

import (
	"context"
	"errors"
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

func (m *mockUserRepository) FindAdminUserIDs() ([]uint, error) {
	args := m.Called()
	return args.Get(0).([]uint), args.Error(1)
}

func (m *mockUserRepository) FindUniversityByUserId(userID uint) (*domain.User, error) {
	args := m.Called(userID)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) FindAllUsers(page, limit int, role, status, search string) ([]domain.User, int64, error) {
	args := m.Called(page, limit, role, status, search)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.User), args.Get(1).(int64), args.Error(2)
}

func (m *mockUserRepository) FindStudentRequest(status string) ([]domain.StudentCardVerification, error) {
	args := m.Called(status)

	if args.Get(0) == nil {
		// return nil ถ้าไม่มีข้อมูล และ กำหนด Error = index 1
		return nil, args.Error(1)
	}

	//return แปลงค่าจาก 0 เป็น type domain.StudentCardVerification และ return พร้อม error
	return args.Get(0).([]domain.StudentCardVerification), args.Error(1)
}

func (m *mockUserRepository) FindUserIDCardRequest(status string) ([]domain.IdCardVerification, error) {
	args := m.Called(status)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]domain.IdCardVerification), args.Error(1)
}

func (m *mockUserRepository) CreateUser(ctx context.Context, usr *domain.User, consent *domain.UserConsent, profile *domain.StudentProfile) (*domain.User, error) {
	args := m.Called(ctx, usr, consent, profile)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
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

func (m *mockUserRepository) SetDefaultBankAccount(userID uint, bankID uint) error {
	args := m.Called(userID, bankID)
	return args.Error(0)
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

func (m *mockUserRepository) FindUser(ctx context.Context, email string) (*domain.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepository) FindUserByResetToken(token string) (*domain.User, error) {
	args := m.Called(token)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepository) FindUserById(id uint) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) != nil {
		return args.Get(0).(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserRepository) FindAllByRole(role string) ([]domain.User, error) {
	args := m.Called(role)
	if args.Get(0) != nil {
		return args.Get(0).([]domain.User), args.Error(1)
	}
	return nil, args.Error(1)
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

func (m *mockAuth) GenerateRefreshToken(userID uint, email string, role string) (string, error) {
	args := m.Called(userID, email, role)
	return args.String(0), args.Error(1)
}

func (m *mockAuth) VerifyRefreshToken(token string) (domain.User, error) {
	args := m.Called(token)
	user, _ := args.Get(0).(domain.User)
	return user, args.Error(1)
}

type mockUniversityRepo struct {
	mock.Mock
}

func (m *mockUniversityRepo) Create(u *domain.University) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *mockUniversityRepo) FindAll() ([]domain.University, error) {
	args := m.Called()
	return args.Get(0).([]domain.University), args.Error(1)
}

func (m *mockUniversityRepo) FindByID(id uint) (*domain.University, error) {
	args := m.Called(id)
	return args.Get(0).(*domain.University), args.Error(1)
}

func (m *mockUniversityRepo) Update(u *domain.University) error {
	args := m.Called(u)
	return args.Error(0)
}

func (m *mockUniversityRepo) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockUniversityRepo) FindByName(nameTH *string, nameEN *string) (*domain.University, error) {
	args := m.Called(nameTH, nameEN)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.University), args.Error(1)
}

func (m *mockUniversityRepo) CreateDomain(d *domain.UniversityDomain) error {
	args := m.Called(d)
	return args.Error(0)
}

func (m *mockUniversityRepo) DeleteDomain(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *mockUniversityRepo) GetUniversityByDomain(ctx context.Context, domainStr string) (*domain.UniversityDomain, error) {
	args := m.Called(ctx, domainStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UniversityDomain), args.Error(1)
}

func (m *mockUniversityRepo) FindDomainByID(ID uint) (*domain.UniversityDomain, error) {
	args := m.Called(ID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.UniversityDomain), args.Error(1)
}

func (m *mockUniversityRepo) UpdateDomain(d *domain.UniversityDomain) error {
	args := m.Called(d)
	return args.Error(0)
}

// ─── mockCache ───────────────────────────────────────────────────────────────

type mockCache struct {
	mock.Mock
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockCache) Del(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	args := m.Called(ctx, key, value, ttl)
	return args.Bool(0), args.Error(1)
}

// newUserServiceWithCache สร้าง services พร้อม cache mock สำหรับ test
func newUserServiceWithCache(repo *mockUserRepository, urepo *mockUniversityRepo, auth *mockAuth, cfg config.AppConfig, notif NotificationService, c *mockCache) UserService {
	return NewUserService(repo, urepo, auth, cfg, notif, c)
}

func TestSignup_Success(t *testing.T) {

	repo := new(mockUserRepository)
	auth := new(mockAuth)
	cache := new(mockCache)

	svc := NewUserService(
		repo,
		nil, // ไม่ใช้ pioneer เลยไม่ต้อง mock
		auth,
		config.AppConfig{
			BaseURL: "http://localhost",
		},
		nil,
		cache,
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

	// SignUp ใช้ cache.Get เช็ค email ก่อน
	cache.On("Get", mock.Anything, "email:test@test.com").Return("", errors.New("cache miss"))
	// cache.Set สำหรับ verify token
	cache.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)

	repo.On("FindUser", mock.Anything, "test@test.com").
		Return(&domain.User{}, gorm.ErrRecordNotFound)

	auth.On("CreateHashedPassword", "password123").
		Return("hashed-password", nil)

	auth.On("GenerateCode").
		Return("verify-token", nil)

	repo.On("CreateUser",
		mock.Anything,
		mock.AnythingOfType("*domain.User"),
		mock.AnythingOfType("*domain.UserConsent"),
		mock.Anything,
	).Return(&domain.User{ID: 1}, nil)

	// run
	msg, err := svc.SignUp(context.Background(), input)

	// assert
	assert.NoError(t, err)
	assert.NotEmpty(t, msg)

	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestGoogleSignin_NewUser_Success(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Mock Google Token Requesty
	httpmock.RegisterResponder("POST", "https://oauth2.googleapis.com/token",
		httpmock.NewStringResponder(200, `{"access_token": "mock_token", "token_type": "Bearer"}`))

	// Mock Google User Info Endpoint
	httpmock.RegisterResponder("GET", "https://www.googleapis.com/oauth2/v2/userinfo",
		httpmock.NewStringResponder(200, `{"id":"123","email":"test@google.com","given_name":"John","family_name":"Doe"}`))

	repo := new(mockUserRepository)
	auth := new(mockAuth)

	svc := NewUserService(repo, nil, auth, config.AppConfig{}, nil, nil)
	oauthConf := &oauth2.Config{
		ClientID: "test", ClientSecret: "test", Endpoint: oauth2.Endpoint{
			TokenURL: "https://oauth2.googleapis.com/token",
		},
	}

	// Define expected DB and Auth actions inside the Service flow
	repo.On("FindUser", mock.Anything, "test@google.com").Return(&domain.User{}, gorm.ErrRecordNotFound)
	repo.On("CreateUser", mock.Anything, mock.AnythingOfType("*domain.User"), mock.AnythingOfType("*domain.UserConsent"), mock.Anything).Return(&domain.User{ID: 1, Email: "test@google.com", Role: "booster"}, nil)
	auth.On("GenerateToken", uint(1), "test@google.com", "booster").Return("mock.jwt.token", nil)

	// Call the method to test
	token, err := svc.GoogleSigning(context.Background(), "mock-code", "booster", oauthConf)

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
	svc := NewUserService(repo, nil, auth, config.AppConfig{}, nil, nil)

	existingUser := &domain.User{
		ID:              2,
		Email:           "test@test.com",
		PasswordHash:    "hashed_password",
		Role:            "pioneer",
		Status:          domain.ACTIVE,
		EmailVerifiedAt: ptr(time.Now()),
	}

	repo.On("FindUser", mock.Anything, "test@test.com").Return(existingUser, nil)
	auth.On("VerifyPassword", "password123", "hashed_password").Return(nil)
	auth.On("GenerateToken", uint(2), "test@test.com", "pioneer").Return("mock.jwt.token", nil)

	token, _, _, err := svc.Signing(context.Background(), "test@test.com", "password123")

	assert.NoError(t, err)
	assert.Equal(t, "mock.jwt.token", token)

	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}

func TestVerifyEmail_Success(t *testing.T) {
	repo := new(mockUserRepository)
	cache := new(mockCache)
	svc := newUserServiceWithCache(repo, nil, nil, config.AppConfig{}, nil, cache)

	token := "123456"
	email := "user@test.com"
	existingUser := &domain.User{
		ID:    3,
		Email: email,
	}

	// cache มี token → return email
	cache.On("Get", mock.Anything, "verify:token:"+token).Return(email, nil)
	cache.On("Del", mock.Anything, "verify:token:"+token).Return(nil)

	repo.On("FindUser", mock.Anything, email).Return(existingUser, nil)
	repo.On("UpdateUser", uint(3), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	msg, err := svc.VerifyEmail(context.Background(), dto.VerifyEmailRequest{Token: token})

	assert.NoError(t, err)
	assert.NotEmpty(t, msg)

	repo.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestVerifyEmail_Fail_InvalidToken(t *testing.T) {
	repo := new(mockUserRepository)
	cache := new(mockCache)
	svc := newUserServiceWithCache(repo, nil, nil, config.AppConfig{}, nil, cache)

	// cache ไม่มี token → error
	cache.On("Get", mock.Anything, "verify:token:bad-token").Return("", errors.New("cache miss"))

	_, err := svc.VerifyEmail(context.Background(), dto.VerifyEmailRequest{Token: "bad-token"})

	assert.Error(t, err)
	assert.Equal(t, "invalid or expired token", err.Error())
	cache.AssertExpectations(t)
}

func TestVerifyEmail_Fail_AlreadyVerified(t *testing.T) {
	repo := new(mockUserRepository)
	cache := new(mockCache)
	svc := newUserServiceWithCache(repo, nil, nil, config.AppConfig{}, nil, cache)

	token := "abc123"
	email := "user@test.com"
	now := time.Now()
	existingUser := &domain.User{
		ID:              3,
		Email:           email,
		EmailVerifiedAt: &now, // already verified
	}

	cache.On("Get", mock.Anything, "verify:token:"+token).Return(email, nil)
	cache.On("Del", mock.Anything, "verify:token:"+token).Return(nil)
	repo.On("FindUser", mock.Anything, email).Return(existingUser, nil)

	_, err := svc.VerifyEmail(context.Background(), dto.VerifyEmailRequest{Token: token})

	assert.Error(t, err)
	assert.Equal(t, "email already verified", err.Error())
	repo.AssertExpectations(t)
	cache.AssertExpectations(t)
}

func TestForgotPassword_Success(t *testing.T) {
	repo := new(mockUserRepository)
	auth := new(mockAuth)
	svc := NewUserService(repo, nil, auth, config.AppConfig{}, nil, nil)

	existingUser := &domain.User{
		ID:     4,
		Email:  "forgot@test.com",
		Status: domain.ACTIVE,
	}

	repo.On("FindUser", mock.Anything, "forgot@test.com").Return(existingUser, nil)
	repo.On("UpdateUser", uint(4), mock.AnythingOfType("map[string]interface {}")).Return(nil)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("POST", "https://api.resend.com/emails", httpmock.NewStringResponder(200, `{}`))

	err := svc.ForgotPassword(context.Background(), "forgot@test.com")

	assert.NoError(t, err)

	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}

func TestSetPassword_Success(t *testing.T) {
	repo := new(mockUserRepository)
	auth := new(mockAuth)
	svc := NewUserService(repo, nil, auth, config.AppConfig{}, nil, nil)

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

// ─── AddBankAccount ─────────────────────────────────────────────────────────

func TestAddBankAccount_Success(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(10)
	accNum := "123456789"

	repo.On("FindUserById", userID).Return(&domain.User{ID: userID}, nil)
	repo.On("FindBankByAccountNumber", accNum).Return((*domain.BankAccount)(nil), nil)
	repo.On("FindBankByUserId", userID).Return([]domain.BankAccount{}, nil)
	repo.On("CreateBankAccount", mock.AnythingOfType("*domain.BankAccount")).Return(nil)
	repo.On("SetDefaultBankAccount", userID, mock.Anything).Return(nil)

	bankName := "Bangkok Bank"
	accName := "John Doe"
	input := dto.BankRequest{
		BankName:      &bankName,
		AccountName:   &accName,
		AccountNumber: &accNum,
	}

	err := svc.AddBankAccount(userID, input)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAddBankAccount_Fail_Duplicate(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(10)
	accNum := "123456789"

	repo.On("FindUserById", userID).Return(&domain.User{ID: userID}, nil)
	// Return existing bank account → duplicate
	repo.On("FindBankByAccountNumber", accNum).Return(&domain.BankAccount{ID: 99, AccountNumber: accNum}, nil)

	bankName := "Bangkok Bank"
	accName := "John Doe"
	input := dto.BankRequest{
		BankName:      &bankName,
		AccountName:   &accName,
		AccountNumber: &accNum,
	}

	err := svc.AddBankAccount(userID, input)

	assert.Error(t, err)
	assert.Equal(t, "account number already exists", err.Error())
	repo.AssertExpectations(t)
}

// ─── GetProfile ─────────────────────────────────────────────────────────────

func TestGetProfile_Success(t *testing.T) {
	repo := new(mockUserRepository)
	cache := new(mockCache)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, cache)

	userID := uint(7)
	expected := &domain.User{ID: userID, Email: "user@test.com", Role: "booster"}

	repo.On("FindUserById", userID).Return(expected, nil)
	cache.On("Set", mock.Anything, "user:7", mock.Anything, 5*time.Minute).Return(nil)

	result, err := svc.GetProfile(context.Background(), userID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user@test.com", result.Email)

	repo.AssertExpectations(t)
}

// ─── VerifyStudent ───────────────────────────────────────────────────────────

func TestVerifyStudent_Success_FirstTime(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(20)
	cardURL := "https://res.cloudinary.com/test/student-card.jpg"
	acceptTerms := true
	declareTruth := true

	// User is pioneer
	repo.On("FindUserById", userID).Return(&domain.User{ID: userID, Role: "pioneer"}, nil)
	// No existing verification
	repo.On("FindLatestStudentVerification", userID).Return((*domain.StudentCardVerification)(nil), nil)
	// Create new verification
	repo.On("CreateStudentVerification", mock.AnythingOfType("*domain.StudentCardVerification")).Return(nil)
	// Create consent records
	repo.On("CreateConsents", mock.AnythingOfType("[]*domain.UserConsent")).Return(nil)

	input := dto.VerifyStudentInput{
		StudentCardURL:     &cardURL,
		AcceptPioneerTerms: &acceptTerms,
		DeclareTruth:       &declareTruth,
	}

	err := svc.VerifyStudent(userID, input)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestVerifyStudent_Fail_AlreadyPending(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(20)
	cardURL := "https://res.cloudinary.com/test/student-card.jpg"
	acceptTerms := true
	declareTruth := true

	repo.On("FindUserById", userID).Return(&domain.User{ID: userID, Role: "pioneer"}, nil)
	// Return existing pending  → should block
	repo.On("FindLatestStudentVerification", userID).Return(
		&domain.StudentCardVerification{ID: 5, Status: domain.VerifyStatusPending}, nil,
	)

	input := dto.VerifyStudentInput{
		StudentCardURL:     &cardURL,
		AcceptPioneerTerms: &acceptTerms,
		DeclareTruth:       &declareTruth,
	}

	err := svc.VerifyStudent(userID, input)

	assert.Error(t, err)
	assert.Equal(t, "verification is already pending", err.Error())
	repo.AssertExpectations(t)
}

func TestVerifyStudent_Fail_NotPioneer(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(20)
	cardURL := "https://res.cloudinary.com/test/student-card.jpg"
	acceptTerms := true
	declareTruth := true

	// User is booster → not allowed
	repo.On("FindUserById", userID).Return(&domain.User{ID: userID, Role: "booster"}, nil)

	input := dto.VerifyStudentInput{
		StudentCardURL:     &cardURL,
		AcceptPioneerTerms: &acceptTerms,
		DeclareTruth:       &declareTruth,
	}

	err := svc.VerifyStudent(userID, input)

	assert.Error(t, err)
	assert.Equal(t, "only pioneer", err.Error())
	repo.AssertExpectations(t)
}

// ─── VerifyID ───────────────────────────────────────────────────────────────

func TestVerifyID_Success_FirstTime(t *testing.T) {
	repo := new(mockUserRepository)
	// IApp OCR will fail → falls back to pending status (still success path)
	svc := NewUserService(repo, nil, nil, config.AppConfig{IAppAPIKey: ""}, nil, nil)

	userID := uint(30)
	idCardURL := "https://res.cloudinary.com/test/idcard.jpg"
	selfieURL := "https://res.cloudinary.com/test/selfie.jpg"
	declareTruth := true

	// No existing verification
	repo.On("FindLatestIdVerification", userID).Return((*domain.IdCardVerification)(nil), nil)
	// Create new verification (pending status since OCR key is empty)
	repo.On("CreateIdVerification", mock.AnythingOfType("*domain.IdCardVerification")).Return(nil)
	// Consent
	repo.On("CreateConsents", mock.AnythingOfType("[]*domain.UserConsent")).Return(nil)

	input := dto.VerifyIDInput{
		IDCardURL:    &idCardURL,
		SelfieURL:    &selfieURL,
		DeclareTruth: &declareTruth,
	}

	err := svc.VerifyID(userID, input)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestVerifyID_Fail_AlreadyApproved(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(30)
	idCardURL := "https://res.cloudinary.com/test/idcard.jpg"
	selfieURL := "https://res.cloudinary.com/test/selfie.jpg"
	declareTruth := true

	// Already approved → must block
	repo.On("FindLatestIdVerification", userID).Return(
		&domain.IdCardVerification{ID: 9, Status: domain.VerifyStatusApproved}, nil,
	)

	input := dto.VerifyIDInput{
		IDCardURL:    &idCardURL,
		SelfieURL:    &selfieURL,
		DeclareTruth: &declareTruth,
	}

	err := svc.VerifyID(userID, input)

	assert.Error(t, err)
	assert.Equal(t, "already verified", err.Error())
	repo.AssertExpectations(t)
}

func TestChangePassword_Success(t *testing.T) {
	repo := new(mockUserRepository)
	auth := new(mockAuth)
	svc := NewUserService(repo, nil, auth, config.AppConfig{}, nil, nil)

	userID := uint(30)
	oldPassword := "Oldpass1!"
	newPassword := "Newpass1!"

	// mock user
	user := &domain.User{
		ID:           userID,
		PasswordHash: "$2a$10$xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", // fake hash
	}

	// mock VerifyPassword (ต้อง return nil = password ถูก)
	auth.On("VerifyPassword", oldPassword, user.PasswordHash).Return(nil)

	// mock find user
	repo.On("FindUserById", userID).Return(user, nil)

	// mock UpdateUser
	repo.On("UpdateUser", userID, mock.Anything).Return(nil)

	// call
	err := svc.ChangePassword(userID, oldPassword, newPassword)

	// assert
	assert.NoError(t, err)

	auth.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestChangePassword_Fail_incorrectOldPassword(t *testing.T) {
	repo := new(mockUserRepository)
	auth := new(mockAuth)
	svc := NewUserService(repo, nil, auth, config.AppConfig{}, nil, nil)

	userID := uint(30)
	oldPassword := "Oldpass11!"
	newPassword := "Newpass1!"

	// mock user
	user := &domain.User{
		ID:           userID,
		PasswordHash: "$2a$10$xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", // fake hash
	}

	// mock VerifyPassword (ต้อง return nil = password ถูก)
	auth.On("VerifyPassword", oldPassword, user.PasswordHash).Return(errors.New("incorrect password"))

	// mock find user
	repo.On("FindUserById", userID).Return(user, nil)

	// call
	err := svc.ChangePassword(userID, oldPassword, newPassword)

	repo.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything)

	// assert
	assert.Error(t, err)

	auth.AssertExpectations(t)
	repo.AssertExpectations(t)
}

func TestChangePassword_Validation(t *testing.T) {
	repo := new(mockUserRepository)
	auth := new(mockAuth)

	svc := NewUserService(repo, nil, auth, config.AppConfig{}, nil, nil)

	userID := uint(1)

	tests := []struct {
		name        string
		newPassword string
		expectError bool
	}{
		{"no uppercase", "newpass1!", true},
		{"no lowercase", "NEWPASS1!", true},
		{"no number", "Newpass!", true},
		{"no special", "Newpass11", true},
		{"valid", "Newpass1!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// ต้อง mock
			user := &domain.User{
				ID:           userID,
				PasswordHash: "hashed",
			}

			repo.On("FindUserById", userID).Return(user, nil)
			auth.On("VerifyPassword", "Oldpass1!", user.PasswordHash).Return(nil)
			repo.On("UpdateUser", userID, mock.Anything).Return(nil)

			err := svc.ChangePassword(userID, "Oldpass1!", tt.newPassword)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAllPendingStatusStudentRequests(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	// mock data
	expected := []domain.StudentCardVerification{
		{ID: 1},
		{ID: 2},
	}

	// ต้อง match "pending"
	repo.On("FindStudentRequest", string(domain.VerifyStatusPending)).
		Return(expected, nil)

	// call
	result, err := svc.GetAllStudentVerifyRequest()

	// assert
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	repo.AssertExpectations(t)
}

func TestGetAllPendingStatusStudentRequests_Error(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	repo.On("FindStudentRequest", string(domain.VerifyStatusPending)).
		Return(nil, errors.New("db error"))

	result, err := svc.GetAllStudentVerifyRequest()

	assert.Error(t, err)
	assert.Nil(t, result)

	repo.AssertExpectations(t)
}

func TestGetAllPendingStatusCardIDRequests(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	// mock data
	expected := []domain.IdCardVerification{
		{ID: 1},
		{ID: 2},
	}

	// ต้อง match "pending"
	repo.On("FindUserIDCardRequest", string(domain.VerifyStatusPending)).
		Return(expected, nil)

	// call
	result, err := svc.GetAllCardIDVerifyRequest()

	// assert
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	repo.AssertExpectations(t)
}

func TestGetAllPendingStatusCardIDRequests_Error(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	repo.On("FindUserIDCardRequest", string(domain.VerifyStatusPending)).
		Return(nil, errors.New("db error"))

	result, err := svc.GetAllCardIDVerifyRequest()

	assert.Error(t, err)
	assert.Nil(t, result)

	repo.AssertExpectations(t)
}

// ─── SuspendUser ─────────────────────────────────────────────────────────────

func TestSuspendUser_Success(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	adminID := uint(1)
	userID := uint(2)

	repo.On("FindUserById", userID).Return(&domain.User{
		ID:     userID,
		Email:  "user@test.com",
		Status: domain.ACTIVE,
	}, nil)
	repo.On("UpdateUser", userID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	err := svc.SuspendUser(adminID, userID, "violates terms")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSuspendUser_Fail_AlreadySuspended(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	adminID := uint(1)
	userID := uint(2)

	repo.On("FindUserById", userID).Return(&domain.User{
		ID:     userID,
		Status: domain.SUSPENDED,
	}, nil)

	err := svc.SuspendUser(adminID, userID, "violates terms")

	assert.Error(t, err)
	assert.Equal(t, "user is already suspended", err.Error())
	repo.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestSuspendUser_Fail_SelfSuspend(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	adminID := uint(1)

	err := svc.SuspendUser(adminID, adminID, "reason")

	assert.Error(t, err)
	assert.Equal(t, "admin cannot be suspended", err.Error())
}

// ─── RollbackActiveUser ───────────────────────────────────────────────────────

func TestRollbackActiveUser_Success(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(5)

	repo.On("FindUserById", userID).Return(&domain.User{
		ID:     userID,
		Status: domain.SUSPENDED,
	}, nil)
	repo.On("UpdateUser", userID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	err := svc.RollbackActiveUser(userID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRollbackActiveUser_Fail_NotSuspended(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(5)

	repo.On("FindUserById", userID).Return(&domain.User{
		ID:     userID,
		Status: domain.ACTIVE,
	}, nil)

	err := svc.RollbackActiveUser(userID)

	assert.Error(t, err)
	assert.Equal(t, "user is not suspended", err.Error())
	repo.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

// ─── ApproveIdCard ────────────────────────────────────────────────────────────

func TestApproveIdCard_Success(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(10)
	adminID := uint(1)

	repo.On("FindUserById", userID).Return(&domain.User{ID: userID}, nil)
	repo.On("FindIdCardStatus", userID).Return(&domain.IdCardVerification{
		ID:     1,
		UserID: userID,
		Status: domain.VerifyStatusPending,
	}, nil)
	repo.On("UpdateIdCardVerification", mock.AnythingOfType("*domain.IdCardVerification")).Return(nil)

	err := svc.ApproveIdCard(userID, adminID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestApproveIdCard_Fail_AlreadyApproved(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(10)
	adminID := uint(1)

	repo.On("FindUserById", userID).Return(&domain.User{ID: userID}, nil)
	repo.On("FindIdCardStatus", userID).Return(&domain.IdCardVerification{
		ID:     1,
		UserID: userID,
		Status: domain.VerifyStatusApproved,
	}, nil)

	err := svc.ApproveIdCard(userID, adminID)

	assert.Error(t, err)
	assert.Equal(t, "student is already verified", err.Error())
	repo.AssertNotCalled(t, "UpdateIdCardVerification", mock.Anything)
	repo.AssertExpectations(t)
}

// ─── RejectStudentCard ────────────────────────────────────────────────────────

func TestRejectStudentCard_Success(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(10)
	adminID := uint(1)

	repo.On("FindUserById", userID).Return(&domain.User{ID: userID}, nil)
	repo.On("FindStudentStatus", userID).Return(&domain.StudentCardVerification{
		ID:     2,
		UserID: userID,
		Status: domain.VerifyStatusPending,
	}, nil)
	repo.On("UpdateStudentCardVerification", mock.AnythingOfType("*domain.StudentCardVerification")).Return(nil)

	err := svc.RejectStudentCard(userID, adminID)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRejectStudentCard_Fail_NoVerification(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(10)
	adminID := uint(1)

	repo.On("FindUserById", userID).Return(&domain.User{ID: userID}, nil)
	repo.On("FindStudentStatus", userID).Return((*domain.StudentCardVerification)(nil), nil)

	err := svc.RejectStudentCard(userID, adminID)

	assert.Error(t, err)
	assert.Equal(t, "no student verification found", err.Error())
	repo.AssertExpectations(t)
}

// ─── SelectRole ───────────────────────────────────────────────────────────────

func TestSelectRole_Success_Booster(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(1)

	repo.On("FindUserById", userID).Return(&domain.User{
		ID:   userID,
		Role: "pending",
	}, nil)
	repo.On("UpdateUser", userID, mock.AnythingOfType("map[string]interface {}")).Return(nil)

	err := svc.SelectRole(context.Background(), userID, "booster")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSelectRole_Fail_AlreadySelected(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	userID := uint(1)

	repo.On("FindUserById", userID).Return(&domain.User{
		ID:   userID,
		Role: "booster", // ไม่ใช่ pending
	}, nil)

	err := svc.SelectRole(context.Background(), userID, "pioneer")

	assert.Error(t, err)
	assert.Equal(t, "user role is already selected", err.Error())
	repo.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestSelectRole_Fail_InvalidRole(t *testing.T) {
	repo := new(mockUserRepository)
	svc := NewUserService(repo, nil, nil, config.AppConfig{}, nil, nil)

	err := svc.SelectRole(context.Background(), 1, "admin")

	assert.Error(t, err)
	assert.Equal(t, "role must be either booster or pioneer", err.Error())
}
