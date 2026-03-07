package service

import (
	"errors"
	"flyup/config"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/repository"
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
	// 1. ตรวจสอบ Password และ Hash
	hPassword, err := s.Auth.CreateHashedPassword(input.Password)
	if err != nil {
		return "", err
	}

	existingUser, err := s.Repo.FindUser(input.Email)

	// ถ้า err เป็น nil แปลว่า เจอข้อมูล -> แสดงว่าอีเมลซ้ำ
	if err == nil && existingUser.ID != 0 {
		return "", errors.New("this email is already registered")
	}

	// ถ้า error ไม่ใช่ user not found แปลว่า DB อาจจะมีปัญหา
	if err != nil && err.Error() != "user not found" {
		return "", errors.New("service temporarily unavailable, please try again later")
	}

	if input.Role == "pioneer" {
		//ดึง Domain ออกจาก Emai
		parts := strings.Split(input.Email, "@")
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
			return "", errors.New("ระบบขัดข้อง กรุณาลองใหม่ภายหลัง")
		}

		// เช็ค Status ของ Domain (ถ้าใน Domain model มี field IsActive)
		if !findDomain.IsActive {
			return "", errors.New("มหาวิทยาลัยนี้ถูกระงับการใช้งานชั่วคราว")
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
		Email:                      input.Email,
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
		ConsentCode: "TERMS_AND_CONDITIONS",
		Accepted:    input.AcceptTerms,
		AcceptedAt:  time.Now(),
	}

	// บันทึก Transaction (User + Consent)
	createdUser, err := s.Repo.CreateUser(newUser, consent)
	if err != nil {
		return "", errors.New("registration failed: " + err.Error())
	}
	log.Printf("User created with ID: %d", createdUser.ID)

	// ส่ง Email โดยใช้ Goroutine
	go func() {
		// เรียกผ่าน s.Auth
		err := s.Auth.SendVerifyEmail(createdUser.Email, token)
		if err != nil {
			log.Printf("Email error: %v", err)
		}
	}()

	return "registration successful, please verify your email", nil
}
