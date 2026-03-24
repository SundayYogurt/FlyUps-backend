package service

import (
	"flyup/config"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type mockUserRepository struct {
	mock.Mock
}

func (m *mockUserRepository) UpdateUser(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *mockUserRepository) FindUser(email string) (*domain.User, error) {
	args := m.Called(email)
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepository) CreateUser(user domain.User, consent domain.UserConsent) (*domain.User, error) {
	args := m.Called(user, consent)
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

	input := dto.UserSignup{
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
		mock.AnythingOfType("domain.User"),
		mock.AnythingOfType("domain.UserConsent"),
	).Return(&domain.User{ID: 1}, nil)

	// run
	msg, err := svc.Signup(input)

	// assert
	assert.NoError(t, err)
	assert.NotEmpty(t, msg)

	repo.AssertExpectations(t)
	auth.AssertExpectations(t)
}
