package service

import (
	"errors"
	"flyup/config"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"flyup/pkg/notification"
	"log"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService interface {
	Signup(input dto.UserSignup) (string, error)
	Signin(email string, password string) (string, error)
	VerifyEmail(input dto.VerifyEmailRequest) (string, error)
	ForgotPassword(email string) error
	SetPassword(token string, newPassword string) error
	GetProfile(userID uint) (*domain.User, error)
}

type userService struct {
	Repo   repository.UserRepository
	URepo  repository.UniversityRepository
	Auth   helper.AuthService
	Config config.AppConfig
}

func NewUserService(
	repo repository.UserRepository,
	urepo repository.UniversityRepository,
	auth helper.AuthService,
	cfg config.AppConfig,
) UserService {
	return &userService{
		Repo:   repo,
		URepo:  urepo,
		Auth:   auth,
		Config: cfg,
	}
}

func (s *userService) Signup(input dto.UserSignup) (string, error) {
	// ตรวจสอบ Password และ Hash
	hPassword, err := s.Auth.CreateHashedPassword(input.Password)
	if err != nil {
		return "", err
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))

	existingUser, err := s.Repo.FindUser(email)

	// ถ้า err เป็น nil แปลว่า เจอข้อมูล -> แสดงว่าอีเมลซ้ำ
	if err == nil && existingUser.ID != 0 {
		return "", errors.New("this email is already registered")
	}

	// ถ้า error ไม่ใช่ user not found แปลว่า DB อาจจะมีปัญหา
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", errors.New("service temporarily unavailable")
	}

	if input.Role == "pioneer" {
		//ดึง Domain ออกจาก Emai
		parts := strings.Split(email, "@")
		if len(parts) < 2 {
			return "", errors.New("invalid email format")
		}
		domainName := parts[1]

		findDomain, err := s.URepo.GetUniversityByDomain(domainName)

		if err != nil {
			// กรณีหา Domain ไม่พบในระบบ (ไม่ใช่ Error ของระบบ แต่เป็น Business Logic)
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", errors.New("sorry!, the domain doesn't exist")
			}
			// กรณี Error อื่นๆ เช่น DB ล่ม
			return "", errors.New("internal server error, try again later")
		}

		// เช็ค Status ของ Domain (ถ้าใน Domain model มี field IsActive)
		if !findDomain.IsActive {
			return "", errors.New("this university is not active")
		}
	}
	token, err := s.Auth.GenerateCode() // จะได้ string ยาว 32 ตัวอักษร
	if err != nil {
		return "", errors.New("internal server error")
	}

	// เตรียม User Domain
	verifyToken := token
	expireAt := time.Now().Add(time.Hour * 24)

	newUser := domain.User{
		Email:                      email,
		PasswordHash:               hPassword,
		FirstName:                  input.FirstName,
		LastName:                   input.LastName,
		Phone:                      input.Phone,
		Role:                       input.Role, // pioneer หรือ booster
		Status:                     "pending",
		VerificationToken:          &verifyToken,
		VerificationTokenExpiresAt: &expireAt,
	}

	// สร้างก้อน Consent จาก AcceptTerms ใน DTO
	consent := domain.UserConsent{
		ConsentCode: domain.ConsentTerm,
		Accepted:    input.AcceptTerms,
		AcceptedAt:  time.Now(),
	}

	// บันทึก Transaction (User + Consent)
	createdUser, err := s.Repo.CreateUser(newUser, consent)
	if err != nil {
		log.Printf("CreateUser error: %v", err)
		return "", errors.New("registration failed")
	}
	log.Printf("User created with ID: %d", createdUser.ID)

	// ส่ง Email โดยใช้ Goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("email panic: %v", r)
			}
		}()
		verifyLink := s.Config.BaseURL + "/verify?token=" + token

		notificationClient := notification.NewNotificationClient(s.Config)

		err := notificationClient.SendVerifyEmail(email, verifyLink)
		if err != nil {
			log.Printf("send verify email error: %v", err)
		}
	}()

	return "registration successful, please verify your email", nil
}

func (s *userService) VerifyEmail(input dto.VerifyEmailRequest) (string, error) {

	user, err := s.Repo.FindUserByVerificationToken(input.Token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("invalid verification token")
		}
		return "", errors.New("internal server error")
	}

	// เช็ค token หมดอายุ
	if user.VerificationTokenExpiresAt == nil ||
		time.Now().After(*user.VerificationTokenExpiresAt) {
		return "", errors.New("verification token expired")
	}

	// เช็ค verify แล้วหรือยัง
	if user.EmailVerifiedAt != nil {
		return "", errors.New("email already verified")
	}

	now := time.Now()

	user.EmailVerifiedAt = &now
	user.VerificationToken = nil
	user.VerificationTokenExpiresAt = nil

	err = s.Repo.UpdateUser(user)
	if err != nil {
		return "", errors.New("failed to verify email")
	}

	return "email verified successfully", nil
}

func (s *userService) findUserByEmail(email string) (*domain.User, error) {
	//perform some db operation
	//business logic
	user, err := s.Repo.FindUser(email)
	return user, err
}

func (s *userService) Signin(email string, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.Repo.FindUser(email)
	if err != nil {
		return "", errors.New("user does not exist with the provided email id")
	}

	if user.EmailVerifiedAt == nil {
		return "", errors.New("please verify your email first")
	}

	err = s.Auth.VerifyPassword(password, user.PasswordHash)

	if err != nil {
		return "", err
	}

	// generate token
	return s.Auth.GenerateToken(user.ID, user.Email, user.Role)
}

func (s *userService) ForgotPassword(email string) error {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := s.Repo.FindUser(email)
	if err != nil || user == nil {
		return nil
	}

	plain, err := helper.GenerateRandomToken(32)
	if err != nil {
		return errors.New("failed to generate reset token")
	}

	hash := helper.Sha256Hex(plain)
	exp := time.Now().Add(30 * time.Minute)

	log.Printf("Reset token (dev only): %s", plain)

	user.ResetTokenHash = &hash
	user.ResetTokenExpiresAt = &exp
	if err := s.Repo.UpdateUser(user); err != nil {
		return errors.New("fail to save user")
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("email panic: %v", r)
			}
		}()
		verifyLink := s.Config.BaseURL + "/reset-password?reset_token=" + plain

		notificationClient := notification.NewNotificationClient(s.Config)

		err := notificationClient.SendResetPasswordEmail(email, verifyLink)
		if err != nil {
			log.Printf("send verify email error: %v", err)
		}
	}()

	return nil
}

func (s *userService) SetPassword(token string, newPassword string) error {

	newPassword = strings.TrimSpace(newPassword)
	token = strings.TrimSpace(token)

	if token == "" || newPassword == "" {
		return errors.New("invalid input")
	}

	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	//check ตัวใหญ่ (A-Z)
	upper := regexp.MustCompile(`[A-Z]`)
	if !upper.MatchString(newPassword) {
		return errors.New("password must contain at least one uppercase letter")
	}

	//check ตัวใหญ่ (A-Z)
	lower := regexp.MustCompile(`[a-z]`)
	if !lower.MatchString(newPassword) {
		return errors.New("password must contain at least one lowercase letter")
	}

	//check ตัวเลข
	digit := regexp.MustCompile(`[0-9]`)
	if !digit.MatchString(newPassword) {
		return errors.New("password must contain at least one number")
	}

	//check อักขระพิเศษ
	special := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)
	if !special.MatchString(newPassword) {
		return errors.New("password must contain at least one special character")
	}

	hash := helper.Sha256Hex(token)

	user, err := s.Repo.FindUserByResetToken(hash)
	if err != nil || user == nil {
		return errors.New("invalid or expired token")
	}

	if user.ResetTokenExpiresAt == nil || time.Now().After(*user.ResetTokenExpiresAt) {
		return errors.New("invalid or expired token")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("fail to hash password")
	}

	user.PasswordHash = string(hashedPassword)

	// invalidate reset token
	user.ResetTokenHash = nil
	user.ResetTokenExpiresAt = nil

	return s.Repo.UpdateUser(user)
}

func (s *userService) GetProfile(userID uint) (*domain.User, error) {
	if userID == 0 {
		return nil, errors.New("invalid user id")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}
