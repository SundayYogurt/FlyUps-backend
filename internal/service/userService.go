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
	"strings"
	"time"

	"gorm.io/gorm"
)

type UserService struct {
	Repo   repository.UserRepository
	URepo  repository.UniversityRepository
	Auth   helper.Auth
	Config config.AppConfig
}

func (s UserService) Signup(input dto.UserSignup) (string, error) {
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

func (s UserService) VerifyEmail(input dto.VerifyEmailRequest) (string, error) {

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

func (s UserService) findUserByEmail(email string) (*domain.User, error) {
	//perform some db operation
	//business logic
	user, err := s.Repo.FindUser(email)
	return user, err
}

func (s UserService) Signin(email string, password string) (string, error) {
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
