package service

import (
	"context"
	"encoding/json"
	"errors"
	"flyup/config"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
	"flyup/internal/port/cache"
	"flyup/internal/repository"
	"flyup/pkg/notification"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService interface {
	SignUp(input dto.UserSignUp) (string, error)
	Signing(email string, password string) (token string, userID uint, role string, err error)
	GoogleSigning(code string, role string, oauthConfig *oauth2.Config) (string, error)
	VerifyEmail(input dto.VerifyEmailRequest) (string, error)
	ForgotPassword(email string) error
	SetPassword(token string, newPassword string) error
	ChangePassword(userID uint, password string, newPassword string) error
	AddPasswordForGoogle(userID uint, newPassword string) error
	GetProfile(userID uint) (*domain.User, error)
	UpdateProfile(userID uint, input dto.ProfileInput) error
	VerifyStudent(userID uint, input dto.VerifyStudentInput) error
	VerifyID(userID uint, input dto.VerifyIDInput) error
	GetAllStudentVerifyRequest() ([]domain.StudentCardVerification, error)
	GetAllCardIDVerifyRequest() ([]domain.IdCardVerification, error)
	AddBankAccount(userID uint, input dto.BankRequest) error
	UpdateBankAccount(userID uint, bankID uint, input dto.BankRequest) error
	FindBankByUserID(id uint) ([]domain.BankAccount, error)
	SetDefaultBankAccount(userID uint, bankID uint) error
	ApproveIdCard(userID uint, adminID uint) error
	ApproveStudentCard(userID uint, adminID uint) error
	RejectIdCard(userID uint, adminID uint) error
	RejectStudentCard(userID uint, adminID uint) error
	SuspendUser(adminID uint, userID uint, reason string) error
	RollbackActiveUser(userID uint) error
	GetNotificationPreferences(userID uint) (map[string]bool, error)
	UpdateNotificationPreferences(userID uint, prefs map[string]bool) error
	ListUser(page, limit int, role, status, search string) ([]domain.User, int64, error)
	CreateUniversity(req dto.CreateUniversityRequest) (*domain.University, error)
	GetAllUniversities() ([]domain.University, error)
	GetUniversityByID(id uint) (*domain.University, error)
	UpdateUniversity(id uint, req dto.CreateUniversityRequest) (*domain.University, error)
	DeleteUniversity(id uint) error
	// CreateDomain domain
	CreateDomain(id uint, req dto.CreateDomainRequest) (*domain.UniversityDomain, error)
	GetUniversityByEmail(email string) (*domain.UniversityDomain, error)
	DeleteDomain(id uint) error
	UpdateDomain(id uint, req dto.UpdateDomainRequest) (*domain.UniversityDomain, error)
	SelectRole(userID uint, newRole string) error
}

type userService struct {
	Repo     repository.UserRepository
	URepo    repository.UniversityRepository
	Auth     helper.AuthService
	Config   config.AppConfig
	NotifSvc NotificationService
	cache    cache.Cache
}

func NewUserService(
	repo repository.UserRepository,
	urepo repository.UniversityRepository,
	auth helper.AuthService,
	cfg config.AppConfig,
	NotifSvc NotificationService,
	cache cache.Cache,
) UserService {
	return &userService{
		Repo:     repo,
		URepo:    urepo,
		Auth:     auth,
		Config:   cfg,
		NotifSvc: NotifSvc,
		cache:    cache,
	}
}

// validatePassword ตรวจสอบ password policy และ hash ให้พร้อมใช้
func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return errors.New("password must contain at least one number")
	}
	if !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password) {
		return errors.New("password must contain at least one special character")
	}
	return nil
}

// hashPassword hash password ด้วย bcrypt
func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", errors.New("fail to hash password")
	}
	return string(hashed), nil
}

// validateAndHashPassword รวม validate + hash ในขั้นตอนเดียว
func validateAndHashPassword(password string) (string, error) {
	if err := validatePassword(password); err != nil {
		return "", err
	}
	return hashPassword(password)
}

// extractDomainFromEmail แยก domain จาก email
func extractDomainFromEmail(email string) (string, error) {
	parts := strings.Split(email, "@")
	if len(parts) < 2 || parts[1] == "" {
		return "", errors.New("invalid email format")
	}
	return parts[1], nil
}

// validatePioneerDomain ตรวจสอบว่า email domain ลงทะเบียนเป็นมหาวิทยาลัยและ active
func (s *userService) validatePioneerDomain(email string) error {
	domainName, err := extractDomainFromEmail(email)
	if err != nil {
		return err
	}
	findDomain, err := s.URepo.GetUniversityByDomain(domainName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("sorry, this email domain is not registered as a university")
		}
		return errors.New("internal server error, try again later")
	}
	if !findDomain.IsActive {
		return errors.New("this university is not active")
	}
	return nil
}

// checkVerificationStatus ตรวจสอบ state ของ verification ว่าสามารถดำเนินการได้หรือไม่
func checkVerificationStatus(status domain.VerifyStatus) error {
	switch status {
	case domain.VerifyStatusApproved:
		return errors.New("student is already verified")
	case domain.VerifyStatusRejected:
		return errors.New("verification was rejected, user must resubmit")
	}
	return nil
}

func (s *userService) ListUser(page, limit int, role, status, search string) ([]domain.User, int64, error) {
	users, total, err := s.Repo.FindAllUsers(page, limit, role, status, search)
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (s *userService) AddPasswordForGoogle(userID uint, newPassword string) error {
	newPassword = strings.TrimSpace(newPassword)

	if userID == 0 {
		return errors.New("invalid user id")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// normal user
	if user.GoogleSub == nil && user.PasswordHash != "" {
		return errors.New("user not have permission for add google password")
	}

	//google user
	if user.GoogleSub != nil && user.PasswordHash != "" {
		return errors.New("user already have password for google password")
	}

	if newPassword == "" {
		return errors.New("invalid newPassword")
	}

	if userID != user.ID {
		return errors.New("user id not match cannot change password")
	}

	hashedPassword, err := validateAndHashPassword(newPassword)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"password_hash": hashedPassword,
	}

	return s.Repo.UpdateUser(user.ID, updates)

}

func (s *userService) SelectRole(userID uint, newRole string) error {
	if userID == 0 {
		return errors.New("invalid user id")
	}

	if newRole != "booster" && newRole != "pioneer" {
		return errors.New("role must be either booster or pioneer")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if user.Role != "pending" {
		return errors.New("user role is already selected")
	}

	if newRole == "pioneer" {
		if err := s.validatePioneerDomain(user.Email); err != nil {
			return err
		}
	}

	updates := map[string]interface{}{
		"role": newRole,
	}

	return s.Repo.UpdateUser(userID, updates)
}

func (s *userService) GetAllStudentVerifyRequest() ([]domain.StudentCardVerification, error) {
	status := domain.VerifyStatusPending
	return s.Repo.FindStudentRequest(string(status))
}

func (s *userService) GetAllCardIDVerifyRequest() ([]domain.IdCardVerification, error) {
	status := domain.VerifyStatusPending
	return s.Repo.FindUserIDCardRequest(string(status))
}

func (s *userService) ChangePassword(userID uint, password string, newPassword string) error {
	newPassword = strings.TrimSpace(newPassword)

	if userID == 0 {
		return errors.New("invalid user id")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if password == "" {
		return errors.New("invalid oldPassword")
	}

	if password == newPassword {
		return errors.New("can't not use old password as new password")
	}

	err = s.Auth.VerifyPassword(password, user.PasswordHash)

	if err != nil {
		return errors.New("password is incorrect")
	}

	if newPassword == "" {
		return errors.New("invalid newPassword")
	}

	if userID != user.ID {
		return errors.New("user id not match cannot change password")
	}

	hashedPassword, err := validateAndHashPassword(newPassword)
	if err != nil {
		return err
	}

	updates := map[string]interface{}{
		"password_hash": hashedPassword,
	}

	return s.Repo.UpdateUser(user.ID, updates)

}

func (s *userService) UpdateDomain(id uint, req dto.UpdateDomainRequest) (*domain.UniversityDomain, error) {
	if id == 0 {
		return nil, errors.New("invalid id")
	}

	// 🔹 หา domain เดิม
	d, err := s.URepo.FindDomainByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("domain not found")
		}
		return nil, errors.New("failed to fetch domain")
	}

	// update domain
	if req.Domain != nil {
		newDomain := strings.ToLower(strings.TrimSpace(*req.Domain))

		if newDomain == "" {
			return nil, errors.New("domain cannot be empty")
		}

		// บังคับ .ac.th
		if !strings.HasSuffix(newDomain, ".ac.th") {
			return nil, errors.New("invalid university domain")
		}

		// เช็คว่าซ้ำกับตัวเองไหม
		if newDomain == d.Domain {
			return nil, errors.New("domain is already this value")
		}

		// เช็คซ้ำใน DB
		existing, _ := s.URepo.GetUniversityByDomain(newDomain)
		if existing != nil && existing.ID != d.ID {
			return nil, errors.New("domain already exists")
		}

		d.Domain = newDomain
	}

	if req.IsActive != nil {
		d.IsActive = *req.IsActive
	}

	if err := s.URepo.UpdateDomain(d); err != nil {
		return nil, errors.New("failed to update domain")
	}

	return d, nil
}

func (s *userService) CreateUniversity(req dto.CreateUniversityRequest) (*domain.University, error) {
	if req.NameTH == nil && req.NameEN == nil {
		return nil, errors.New("name th or name_en required")
	}

	if req.Province == nil {
		return nil, errors.New("province required")
	}

	existing, err := s.URepo.FindByName(req.NameTH, req.NameEN)
	if err == nil && existing != nil {
		return nil, errors.New("university already exists")
	}

	// ถ้า error ที่ไม่ใช่ not found
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("failed to check existing university")
	}

	u := domain.University{
		NameTH:   req.NameTH,
		NameEN:   req.NameEN,
		Province: req.Province,
	}

	if err := s.URepo.Create(&u); err != nil {
		return nil, errors.New("failed to create university")
	}

	return &u, nil
}

func (s *userService) GetAllUniversities() ([]domain.University, error) {
	return s.URepo.FindAll()
}

func (s *userService) GetUniversityByID(id uint) (*domain.University, error) {
	if id == 0 {
		return nil, errors.New("invalid id")
	}

	return s.URepo.FindByID(id)
}

func (s *userService) UpdateUniversity(id uint, req dto.CreateUniversityRequest) (*domain.University, error) {
	u, err := s.URepo.FindByID(id)
	if err != nil {
		return nil, errors.New("university not found")
	}

	if req.NameTH != nil {
		u.NameTH = req.NameTH
	}
	if req.NameEN != nil {
		u.NameEN = req.NameEN
	}
	if req.Province != nil {
		u.Province = req.Province
	}

	// check NameTH
	if req.NameTH != nil {
		newNameTH := strings.TrimSpace(*req.NameTH)
		if newNameTH == "" {
			return nil, errors.New("name_th cannot be empty")
		}

		// เช็คซ้ำด้วย FindByName
		existing, _ := s.URepo.FindByName(&newNameTH, nil)
		if existing != nil && existing.ID != u.ID {
			return nil, errors.New("name_th already exists")
		}

		u.NameTH = &newNameTH
	}

	// check NameEN
	if req.NameEN != nil {
		newNameEN := strings.TrimSpace(*req.NameEN)
		if newNameEN == "" {
			return nil, errors.New("name_en cannot be empty")
		}

		// เช็คซ้ำด้วย FindByName
		existing, _ := s.URepo.FindByName(nil, &newNameEN)
		if existing != nil && existing.ID != u.ID {
			return nil, errors.New("name_en already exists")
		}

		u.NameEN = &newNameEN
	}

	if err := s.URepo.Update(u); err != nil {
		return nil, errors.New("failed to update university")
	}

	return u, nil
}

func (s *userService) DeleteUniversity(id uint) error {
	if id == 0 {
		return errors.New("invalid id")
	}
	return s.URepo.Delete(id)
}

func (s *userService) CreateDomain(id uint, req dto.CreateDomainRequest) (*domain.UniversityDomain, error) {
	if id == 0 || req.Domain == "" {
		return nil, errors.New("missing required fields")
	}

	// normalize domain ก่อนเช็ค
	domainStr := strings.ToLower(strings.TrimSpace(req.Domain))
	if !strings.Contains(domainStr, ".ac.th") {
		return nil, errors.New("invalid university domain")
	}

	// เช็คว่ามี domain อยู่แล้วหรือไม่
	existing, err := s.URepo.GetUniversityByDomain(domainStr)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("failed to check existing domain")
	}
	if existing != nil {
		return nil, errors.New("domain is already used")
	}

	// เช็คว่า university ที่จะเพิ่ม domain มีอยู่ไหม
	_, err = s.URepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("university not found")
		}
		return nil, errors.New("failed to fetch university")
	}

	// สร้าง domain ใหม่
	d := domain.UniversityDomain{
		UniversityID: id,
		Domain:       domainStr,
		IsActive:     true,
	}

	if err := s.URepo.CreateDomain(&d); err != nil {
		return nil, errors.New("failed to create domain")
	}

	return &d, nil
}

func (s *userService) GetUniversityByEmail(email string) (*domain.UniversityDomain, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return nil, errors.New("invalid email format")
	}

	domainPart := parts[1]

	return s.URepo.GetUniversityByDomain(domainPart)
}

func (s *userService) DeleteDomain(id uint) error {
	if id == 0 {
		return errors.New("invalid id")
	}

	_, err := s.URepo.FindDomainByID(id)
	if err != nil {
		return errors.New("domain not found")
	}

	return s.URepo.DeleteDomain(id)
}

func (s *userService) RollbackActiveUser(userID uint) error {
	if userID == 0 {
		return errors.New("invalid userId")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if user.Status == domain.ACTIVE {
		return errors.New("user is not suspended")
	}

	updates := map[string]interface{}{
		"status":         domain.ACTIVE,
		"suspend_reason": nil,
		"suspended_at":   nil,
		"suspended_by":   nil,
	}

	return s.Repo.UpdateUser(userID, updates)
}

func (s *userService) GetNotificationPreferences(userID uint) (map[string]bool, error) {
	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return nil, err
	}
	if user.NotificationPreferences == nil {
		return map[string]bool{}, nil
	}
	return map[string]bool(user.NotificationPreferences), nil
}

func (s *userService) UpdateNotificationPreferences(userID uint, prefs map[string]bool) error {
	jsonBytes, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	return s.Repo.UpdateUser(userID, map[string]interface{}{
		"notification_preferences": string(jsonBytes),
	})
}

func (s *userService) SuspendUser(adminID uint, userID uint, reason string) error {
	if userID == 0 {
		return errors.New("invalid userId")
	}

	if adminID == 0 {
		return errors.New("invalid adminId")
	}

	if reason == "" {
		return errors.New("invalid reason")
	}

	if userID == adminID {
		return errors.New("admin cannot be suspended")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if user.Status == domain.SUSPENDED {
		return errors.New("user is already suspended")
	}

	now := time.Now()

	updates := map[string]interface{}{
		"status":         domain.SUSPENDED,
		"suspend_reason": reason,
		"suspended_at":   now,
		"suspended_by":   adminID,
	}

	email := user.Email

	go func(email, reason string) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("email panic: %v", r)
			}
		}()

		notificationClient := notification.NewNotificationClient(s.Config)

		err := notificationClient.SendUserSuspendedEmail(email, reason)
		if err != nil {
			log.Printf("send verify email error: %v", err)
		}
	}(email, reason)

	return s.Repo.UpdateUser(userID, updates)

}

func (s *userService) RejectIdCard(userID uint, adminID uint) error {
	if userID == 0 {
		return errors.New("invalid id")
	}

	if adminID == 0 {
		return errors.New("invalid adminId")
	}

	_, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	student, err := s.Repo.FindIdCardStatus(userID)
	if err != nil {
		return errors.New("failed to fetch id card verification")
	}

	if student == nil {
		return errors.New("no student verification found")
	}

	if err := checkVerificationStatus(student.Status); err != nil {
		return err
	}

	v := &domain.IdCardVerification{
		UserID:     userID,
		Status:     domain.VerifyStatusRejected,
		ReviewedBy: &adminID,
	}

	if err := s.Repo.UpdateIdCardVerification(v); err != nil {
		return errors.New("failed to update id card verification")
	}

	if s.NotifSvc != nil {
		s.NotifSvc.CreateAndPush(userID, domain.NotifVerificationRejected,
			"บัตรประชาชนถูกปฏิเสธ",
			"บัตรประชาชนของคุณไม่ผ่านการตรวจสอบ กรุณาอัปโหลดใหม่",
			nil, nil,
		)
	}

	return nil
}

func (s *userService) RejectStudentCard(userID uint, adminID uint) error {
	if userID == 0 {
		return errors.New("invalid id")
	}

	if adminID == 0 {
		return errors.New("invalid adminId")
	}

	_, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	student, err := s.Repo.FindStudentStatus(userID)
	if err != nil {
		return errors.New("failed to fetch student card verification")
	}

	if student == nil {
		return errors.New("no student verification found")
	}

	if err := checkVerificationStatus(student.Status); err != nil {
		return err
	}

	v := &domain.StudentCardVerification{
		UserID:     userID,
		Status:     domain.VerifyStatusRejected,
		ReviewedBy: &adminID,
	}

	if err := s.Repo.UpdateStudentCardVerification(v); err != nil {
		return errors.New("failed to update student card verification")
	}

	if s.NotifSvc != nil {
		s.NotifSvc.CreateAndPush(userID, domain.NotifVerificationRejected,
			"บัตรนักศึกษาถูกปฏิเสธ",
			"บัตรนักศึกษาของคุณไม่ผ่านการตรวจสอบ กรุณาอัปโหลดใหม่",
			nil, nil,
		)
	}

	return nil
}

func (s *userService) ApproveStudentCard(userID uint, adminID uint) error {
	if userID == 0 {
		return errors.New("invalid id")
	}

	if adminID == 0 {
		return errors.New("invalid adminId")
	}

	_, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	student, err := s.Repo.FindStudentStatus(userID)
	if err != nil {
		return errors.New("failed to fetch student card verification")
	}

	if student == nil {
		return errors.New("no student verification found")
	}

	if err := checkVerificationStatus(student.Status); err != nil {
		return err
	}

	now := time.Now()

	v := &domain.StudentCardVerification{
		UserID:     userID,
		Status:     domain.VerifyStatusApproved,
		VerifiedAt: &now,
		ReviewedBy: &adminID,
	}

	if err := s.Repo.UpdateStudentCardVerification(v); err != nil {
		return errors.New("failed to update student card verification")
	}

	if s.NotifSvc != nil {
		s.NotifSvc.CreateAndPush(userID, domain.NotifVerificationApproved,
			"บัตรนักศึกษาอนุมัติแล้ว",
			"บัตรนักศึกษาของคุณได้รับการอนุมัติเรียบร้อยแล้ว",
			nil, nil,
		)
	}

	return nil
}

func (s *userService) ApproveIdCard(userID uint, adminID uint) error {
	if userID == 0 {
		return errors.New("invalid id")
	}

	if adminID == 0 {
		return errors.New("invalid adminId")
	}

	_, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	student, err := s.Repo.FindIdCardStatus(userID)
	if err != nil {
		return errors.New("failed to fetch id card verification")
	}

	if student == nil {
		return errors.New("no student verification found")
	}

	if err := checkVerificationStatus(student.Status); err != nil {
		return err
	}

	now := time.Now()

	v := &domain.IdCardVerification{
		UserID:     userID,
		Status:     domain.VerifyStatusApproved,
		VerifiedAt: &now,
		ReviewedBy: &adminID,
	}

	if err := s.Repo.UpdateIdCardVerification(v); err != nil {
		return errors.New("failed to update id card verification")
	}

	if s.NotifSvc != nil {
		s.NotifSvc.CreateAndPush(userID, domain.NotifVerificationApproved,
			"บัตรประชาชนอนุมัติแล้ว",
			"บัตรประชาชนของคุณได้รับการอนุมัติเรียบร้อยแล้ว",
			nil, nil,
		)
	}

	return nil
}

func (s *userService) FindBankByUserID(userID uint) ([]domain.BankAccount, error) {
	if userID == 0 {
		return nil, errors.New("invalid id")
	}

	userBank, err := s.Repo.FindBankByUserId(userID)
	if err != nil {
		return nil, errors.New("failed to fetch bank accounts")
	}

	if len(userBank) == 0 {
		return nil, errors.New("bank not found")
	}

	return userBank, nil
}

func (s *userService) SetDefaultBankAccount(userID uint, bankID uint) error {
	if userID == 0 || bankID == 0 {
		return errors.New("invalid id")
	}

	// เช็คว่าเป็นเจ้าของบัญชีจริงไหม
	bank, err := s.Repo.FindBankById(bankID)
	if err != nil {
		return errors.New("bank not found")
	}
	if bank == nil || bank.UserID != userID {
		return errors.New("not your bank account")
	}

	return s.Repo.SetDefaultBankAccount(userID, bankID)
}

func (s *userService) UpdateBankAccount(userID uint, bankID uint, input dto.BankRequest) error {
	if userID == 0 || bankID == 0 {
		return errors.New("invalid id")
	}
	// 1. หา bank
	bank, err := s.Repo.FindBankById(bankID)
	if err != nil {
		return errors.New("bank not found")
	}
	if bank == nil {
		return errors.New("bank not found")
	}

	// 2. เช็ค ownership
	if bank.UserID != userID {
		return errors.New("not your bank account")
	}

	// 3. validate + normalize
	if input.BankName != nil {
		name := strings.TrimSpace(*input.BankName)
		if name == "" {
			return errors.New("bank name is required")
		}
		bank.BankName = name
	}

	if input.AccountName != nil {
		name := strings.TrimSpace(*input.AccountName)
		if name == "" {
			return errors.New("account name is required")
		}
		bank.AccountName = name
	}

	if input.AccountNumber != nil {
		number := strings.TrimSpace(*input.AccountNumber)

		if number == "" {
			return errors.New("account number is required")
		}

		if _, err := strconv.Atoi(number); err != nil {
			return errors.New("account number must be numeric")
		}

		if len(number) < 8 || len(number) > 15 {
			return errors.New("invalid account number length")
		}

		bank.AccountNumber = number
	}

	// 4. default logic
	if input.IsDefault != nil && *input.IsDefault {
		if err := s.Repo.SetDefaultBankAccount(userID, bankID); err != nil {
			return errors.New("failed to set default bank account")
		}
		bank.IsDefault = true
	}

	// 5. save
	return s.Repo.UpdateBankAccount(bank)
}

func (s *userService) AddBankAccount(userID uint, input dto.BankRequest) error {
	if userID == 0 {
		return errors.New("invalid user ID")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}
	if user == nil {
		return errors.New("user not found")
	}

	if input.BankName == nil || strings.TrimSpace(*input.BankName) == "" {
		return errors.New("bank name is required")
	}

	if input.AccountName == nil || strings.TrimSpace(*input.AccountName) == "" {
		return errors.New("account name is required")
	}

	if input.AccountNumber == nil || strings.TrimSpace(*input.AccountNumber) == "" {
		return errors.New("account number is required")
	}

	if _, err := strconv.Atoi(*input.AccountNumber); err != nil {
		return errors.New("account number must be numeric")
	}

	if len(*input.AccountNumber) < 8 || len(*input.AccountNumber) > 15 {
		return errors.New("invalid account number length")
	}

	existing, err := s.Repo.FindBankByAccountNumber(*input.AccountNumber)
	if err != nil {
		return errors.New("failed to check existing account number")
	}

	if existing != nil {
		return errors.New("account number already exists")
	}

	// check if this is the first bank account
	banks, _ := s.Repo.FindBankByUserId(userID)
	isDefault := false
	if len(banks) == 0 {
		isDefault = true // first account is always default
	} else if input.IsDefault != nil && *input.IsDefault {
		isDefault = true
	}

	bank := &domain.BankAccount{
		UserID:        userID,
		BankName:      *input.BankName,
		AccountName:   *input.AccountName,
		AccountNumber: *input.AccountNumber,
		IsDefault:     isDefault,
	}

	if isDefault {
		// If we are setting this as default, we need to clear others first (handled by transaction if we use Repo method, but here we are creating)
		// Better to use the same transaction logic.
		err := s.Repo.CreateBankAccount(bank)
		if err != nil {
			return err
		}
		return s.Repo.SetDefaultBankAccount(userID, bank.ID)
	}

	return s.Repo.CreateBankAccount(bank)

}

func (s *userService) SignUp(input dto.UserSignUp) (string, error) {
	ctx := context.Background()

	email := strings.ToLower(strings.TrimSpace(input.Email))
	emailKey := "email:" + email

	cached, err := s.cache.Get(ctx, emailKey)
	if err == nil && cached != "" {
		return "", errors.New("this email is already registered")
	}

	existingUser, err := s.Repo.FindUser(email)
	if err == nil && existingUser.ID != 0 {
		_ = s.cache.Set(ctx, emailKey, "1", 10*time.Minute)
		return "", errors.New("this email is already registered")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", errors.New("service temporarily unavailable")
	}

	hPassword, err := s.Auth.CreateHashedPassword(input.Password)
	if err != nil {
		return "", err
	}

	var findDomain *domain.UniversityDomain

	if input.Role == "pioneer" {
		parts := strings.Split(email, "@")
		if len(parts) < 2 {
			return "", errors.New("invalid email format")
		}

		domainName := parts[1]
		domainKey := "university:domain:" + domainName

		cachedUni, err := s.cache.Get(ctx, domainKey)
		if err == nil && cachedUni != "" {
			_ = json.Unmarshal([]byte(cachedUni), &findDomain)
		} else {
			findDomain, err = s.URepo.GetUniversityByDomain(domainName)
			if err != nil {
				return "", errors.New("domain not found")
			}

			data, _ := json.Marshal(findDomain)
			_ = s.cache.Set(ctx, domainKey, string(data), 24*time.Hour)
		}

		if !findDomain.IsActive {
			return "", errors.New("this university is not active")
		}
	}

	token, err := s.Auth.GenerateCode()
	if err != nil {
		return "", errors.New("internal server error")
	}

	verifyKey := "verify:token:" + token
	_ = s.cache.Set(ctx, verifyKey, email, 24*time.Hour)

	verifyToken := token
	expireAt := time.Now().Add(24 * time.Hour)

	newUser := &domain.User{
		Email:                      email,
		PasswordHash:               hPassword,
		FirstName:                  input.FirstName,
		LastName:                   input.LastName,
		Phone:                      input.Phone,
		Role:                       input.Role,
		Status:                     domain.ACTIVE,
		VerificationToken:          &verifyToken,
		VerificationTokenExpiresAt: &expireAt,
	}

	consent := &domain.UserConsent{
		ConsentCode: domain.ConsentTerm,
		Accepted:    input.AcceptTerms,
		AcceptedAt:  time.Now(),
	}

	_, err = s.Repo.CreateUser(newUser, consent)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") {
			return "", errors.New("this email is already registered")
		}
		return "", errors.New("registration failed")
	}

	go func() {
		verifyLink := strings.TrimRight(s.Config.BaseURL, "/") + "/verify?token=" + token
		notificationClient := notification.NewNotificationClient(s.Config)
		_ = notificationClient.SendVerifyEmail(email, verifyLink)
	}()

	return "registration successful, please verify your email", nil
}

func (s *userService) VerifyEmail(input dto.VerifyEmailRequest) (string, error) {
	ctx := context.Background()

	key := "verify:token:" + input.Token

	email, err := s.cache.Get(ctx, key)
	if err != nil {
		return "", errors.New("invalid or expired token")
	}

	_ = s.cache.Del(ctx, key)

	user, err := s.Repo.FindUser(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("user not found")
		}
		return "", errors.New("internal server error")
	}

	if user.EmailVerifiedAt != nil {
		return "", errors.New("email already verified")
	}

	now := time.Now()

	updates := map[string]interface{}{
		"email_verified_at":             now,
		"verification_token":            nil,
		"verification_token_expires_at": nil,
	}

	if err := s.Repo.UpdateUser(user.ID, updates); err != nil {
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

func (s *userService) Signing(email string, password string) (string, uint, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.Repo.FindUser(email)
	if err != nil {
		return "", 0, "", errors.New("invalid email or password")
	}
	if user.EmailVerifiedAt == nil {
		return "", 0, "", errors.New("please verify your email first")
	}
	if user.Status == domain.SUSPENDED {
		return "", 0, "", errors.New("your account has been suspended")
	}
	if user.GoogleSub != nil && user.PasswordHash == "" {
		return "", 0, "", errors.New("this email is registered with Google. Please use 'Continue with Google' to login")
	}
	if err = s.Auth.VerifyPassword(password, user.PasswordHash); err != nil {
		return "", 0, "", errors.New("invalid email or password")
	}
	token, err := s.Auth.GenerateToken(user.ID, user.Email, user.Role)
	return token, user.ID, user.Role, err
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

	updates := map[string]interface{}{
		"reset_token_hash":       hash,
		"reset_token_expires_at": exp,
	}

	if err := s.Repo.UpdateUser(user.ID, updates); err != nil {
		return errors.New("fail to save user")
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("email panic: %v", r)
			}
		}()
		verifyLink := strings.TrimRight(s.Config.BaseURL, "/") + "/reset-password?reset_token=" + plain

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

	hashedPassword, err := validateAndHashPassword(newPassword)
	if err != nil {
		return err
	}

	hash := helper.Sha256Hex(token)

	user, err := s.Repo.FindUserByResetToken(hash)
	if err != nil || user == nil {
		return errors.New("invalid or expired token")
	}

	if user.ResetTokenExpiresAt == nil || time.Now().After(*user.ResetTokenExpiresAt) {
		return errors.New("invalid or expired token")
	}

	updates := map[string]interface{}{
		"password_hash":          hashedPassword,
		"reset_token_hash":       nil,
		"reset_token_expires_at": nil,
	}

	return s.Repo.UpdateUser(user.ID, updates)
}

func (s *userService) GetProfile(userID uint) (*domain.User, error) {
	if userID == 0 {
		return nil, errors.New("invalid user id")
	}

	ctx := context.Background()
	key := "user:" + strconv.Itoa(int(userID))

	// ดึงจาก DB เสมอเพื่อให้ HasPassword ถูกต้อง
	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		// ถ้า DB fail ลอง fallback จาก cache
		val, cacheErr := s.cache.Get(ctx, key)
		if cacheErr == nil {
			var cachedUser domain.User
			if jsonErr := json.Unmarshal([]byte(val), &cachedUser); jsonErr == nil {
				log.Println("DB failed, Cache Fallback")
				return &cachedUser, nil
			}
		}
		return nil, errors.New("failed to fetch user")
	}

	// Set HasPassword จาก DB โดยตรง (ถูกต้องเสมอ)
	hasPassword := user.PasswordHash != ""
	user.HasPassword = &hasPassword

	// ดึง university จาก university_domains ตาม email domain ของ user
	parts := strings.Split(user.Email, "@")
	if len(parts) == 2 && s.URepo != nil {
		uniDomain, err := s.URepo.GetUniversityByDomain(parts[1])
		if err == nil && uniDomain != nil {
			if user.StudentProfile == nil {
				user.StudentProfile = &domain.StudentProfile{}
			}
			user.StudentProfile.University = &uniDomain.University
		}
	}

	// Cache ไว้สำหรับ fallback (ไม่ใช้เป็น primary source)
	data, _ := json.Marshal(user)
	s.cache.Set(ctx, key, data, 5*time.Minute)

	return user, nil
}

func (s *userService) GoogleSigning(code string, role string, oauthConfig *oauth2.Config) (string, error) {
	// 1. Exchange custom code for an access token
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return "", errors.New("failed to exchange token: " + err.Error())
	}

	// 2. Fetch user details from Google
	resp, err := oauthConfig.Client(context.Background(), token).Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return "", errors.New("failed to fetch user info: " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("failed to read user info: " + err.Error())
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Picture       string `json:"picture"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
	}

	if err := json.Unmarshal(body, &googleUser); err != nil {
		return "", errors.New("failed to parse user info: " + err.Error())
	}

	// 3. Validate Pioneer Email Domain BEFORE checking if user exists
	if role == "pioneer" {
		if err := s.validatePioneerDomain(googleUser.Email); err != nil {
			return "", err
		}
	}

	// 4. Check if user exists by email
	user, err := s.Repo.FindUser(googleUser.Email)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("internal server error")
		}

		// User not found, automatically register them!
		if role != "pioneer" && role != "booster" {
			role = "pending" // Fallback Default
		}

		googleSub := googleUser.ID
		now := time.Now()
		newUser := &domain.User{
			Email:           googleUser.Email,
			FirstName:       googleUser.GivenName,
			LastName:        googleUser.FamilyName,
			Role:            role,
			Status:          domain.ACTIVE,
			GoogleSub:       &googleSub,
			EmailVerifiedAt: &now,
		}

		consent := &domain.UserConsent{
			ConsentCode: domain.ConsentTerm,
			Accepted:    true,
			AcceptedAt:  now,
		}

		user, err = s.Repo.CreateUser(newUser, consent)
		if err != nil {
			return "", errors.New("failed to create user")
		}
	} else {
		// User exists, just ensure their GoogleSub is updated if missing
		if user.GoogleSub == nil || *user.GoogleSub != googleUser.ID {
			googleSub := googleUser.ID
			s.Repo.UpdateUser(user.ID, map[string]interface{}{
				"google_sub":        &googleSub,
				"email_verified_at": time.Now(),
			})
		}
	}

	// Check if user is suspended
	if user.Status == domain.SUSPENDED {
		return "", errors.New("your account has been suspended")
	}

	// 4. Generate JWT Token
	return s.Auth.GenerateToken(user.ID, user.Email, user.Role)
}

func (s *userService) UpdateProfile(userID uint, input dto.ProfileInput) error {
	// 1. Validate userID
	if userID == 0 {
		return errors.New("invalid user ID")
	}

	// 2. Find user
	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	// 3. Update normal profile
	updates := map[string]interface{}{}
	if input.FirstName != nil {
		updates["first_name"] = strings.TrimSpace(*input.FirstName)
	}
	if input.LastName != nil {
		updates["last_name"] = strings.TrimSpace(*input.LastName)
	}
	if input.Phone != nil {
		updates["phone"] = strings.TrimSpace(*input.Phone)
	}
	if input.Address != nil {
		addr := strings.TrimSpace(*input.Address)
		updates["address"] = &addr
	}

	if input.Picture != nil {
		updates["picture"] = strings.TrimSpace(*input.Picture)
	}

	// 4. Pioneer-specific update
	var studentProfile *domain.StudentProfile
	if user.Role == "pioneer" {
		// University ต้องมาจาก domain ของ email ที่สมัครเท่านั้น
		parts := strings.Split(strings.ToLower(strings.TrimSpace(user.Email)), "@")
		if len(parts) < 2 {
			return errors.New("invalid email for university lookup")
		}
		domainName := strings.TrimSpace(parts[1])

		uniDomain, err := s.URepo.GetUniversityByDomain(domainName)
		if err != nil || uniDomain == nil || uniDomain.UniversityID == 0 {
			return errors.New("university not found for this email domain")
		}
		universityID := uniDomain.UniversityID
		log.Printf("[UpdateProfile] derived university_id=%d from domain=%s (user_id=%d)", universityID, domainName, userID)

		// Build student profile payload (explicit upsert)
		studentProfile = &domain.StudentProfile{
			UserID:       userID,
			UniversityID: universityID,
		}

		// Update optional student fields
		if input.Faculty != nil {
			faculty := strings.TrimSpace(*input.Faculty)
			studentProfile.Faculty = &faculty
		}
		if input.Major != nil {
			major := strings.TrimSpace(*input.Major)
			studentProfile.Major = &major
		}
		if input.Bio != nil {
			bio := strings.TrimSpace(*input.Bio)
			studentProfile.Bio = &bio
		}
		if input.Portfolio != nil {
			portfolio := strings.TrimSpace(*input.Portfolio)
			studentProfile.Portfolio = &portfolio
		}
		if input.Skills != nil {
			skills := strings.TrimSpace(*input.Skills)
			studentProfile.Skills = &skills
		}
		if input.StudentCode != nil {
			code := strings.TrimSpace(*input.StudentCode)
			studentProfile.StudentCode = &code
		}
	}

	// 5. Save explicitly (avoid GORM association autosave pitfalls)
	log.Printf("[UpdateProfile] applying explicit profile update (user_id=%d)", userID)
	if len(updates) > 0 {
		if err := s.Repo.UpdateUser(userID, updates); err != nil {
			return errors.New("failed to update profile")
		}
	}
	if studentProfile != nil {
		if err := s.Repo.UpsertStudentProfileByUserID(studentProfile); err != nil {
			return errors.New("failed to update student profile")
		}
	}
	return nil
}

func (s *userService) VerifyStudent(userID uint, input dto.VerifyStudentInput) error {

	if userID == 0 {
		return errors.New("invalid user ID")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return errors.New("user not found")
	}

	if user.Role != "pioneer" {
		return errors.New("only pioneer")
	}

	if input.StudentCardURL == nil {
		return errors.New("student card required")
	}

	if input.AcceptPioneerTerms == nil || !*input.AcceptPioneerTerms {
		return errors.New("must accept terms")
	}

	if input.DeclareTruth == nil || !*input.DeclareTruth {
		return errors.New("must confirm information is true")
	}

	// หา record ล่าสุด
	existing, err := s.Repo.FindLatestStudentVerification(userID)
	if err != nil {
		return errors.New("failed to fetch student verification")
	}

	// กัน state
	if existing != nil {
		switch existing.Status {
		case domain.VerifyStatusPending:
			return errors.New("verification is already pending")

		case domain.VerifyStatusApproved:
			return errors.New("already verified")
		}
	}

	verify := &domain.StudentCardVerification{
		UserID:   userID,
		Document: *input.StudentCardURL,
		Status:   domain.VerifyStatusPending,
	}

	consents := []*domain.UserConsent{
		{
			UserID:      userID,
			ConsentCode: domain.ConsentPioneerTerm,
			Accepted:    true,
			AcceptedAt:  time.Now(),
		},
		{
			UserID:      userID,
			ConsentCode: domain.ConsentDeclareTruth,
			Accepted:    true,
			AcceptedAt:  time.Now(),
		},
	}

	// create vs update
	if existing == nil {
		// ครั้งแรก
		if err := s.Repo.CreateStudentVerification(verify); err != nil {
			return errors.New("failed to create student verification")
		}
	} else {
		// rejected → update
		verify.ID = existing.ID
		if err := s.Repo.UpdateStudentVerification(verify); err != nil {
			return errors.New("failed to update student verification")
		}
	}

	return s.Repo.CreateConsents(consents)
}

func (s *userService) VerifyID(userID uint, input dto.VerifyIDInput) error {

	if userID == 0 {
		return errors.New("invalid user ID")
	}

	if input.IDCardURL == nil {
		return errors.New("ID card required")
	}

	if input.SelfieURL == nil {
		return errors.New("selfie image required")
	}

	if input.DeclareTruth == nil || !*input.DeclareTruth {
		return errors.New("must confirm information is true")
	}

	// หา record ล่าสุด
	existing, err := s.Repo.FindLatestIdVerification(userID)
	if err != nil {
		return errors.New("failed to fetch id verification")
	}

	// กัน state
	if existing != nil {
		switch existing.Status {
		case domain.VerifyStatusPending:
			return errors.New("verification is already pending")

		case domain.VerifyStatusApproved:
			return errors.New("already verified")
		}
	}

	iappSvc := helper.NewIAppService(s.Config.IAppAPIKey)

	ocrPayload, err := iappSvc.VerifyFaceAndIDCard(*input.IDCardURL, *input.SelfieURL)

	// default = pending
	finalStatus := domain.VerifyStatusPending
	var verifiedAt *time.Time
	var faceScore *float64

	// fallback ถ้า OCR พัง
	if err != nil {
		log.Printf("[VerifyID] OCR error: %v", err)
	} else {

		var iAppResult struct {
			Total struct {
				IsSamePerson string  `json:"isSamePerson"`
				Confidence   float64 `json:"confidence"`
			} `json:"total"`
		}

		if err := json.Unmarshal([]byte(ocrPayload), &iAppResult); err != nil {
			log.Printf("[VerifyID] JSON parse error: %v", err)
		} else {
			conf := iAppResult.Total.Confidence
			faceScore = &conf

			if iAppResult.Total.IsSamePerson == "true" && conf >= 80.0 {
				finalStatus = domain.VerifyStatusApproved
				now := time.Now()
				verifiedAt = &now
			}
		}
	}

	// สร้าง object
	verify := &domain.IdCardVerification{
		UserID:     userID,
		Document:   *input.IDCardURL,
		SelfieURL:  input.SelfieURL,
		Status:     finalStatus,
		FaceScore:  faceScore,
		OcrPayload: &ocrPayload,
		VerifiedAt: verifiedAt,
	}

	// ตัดสินใจ: create vs update
	if existing == nil {
		// create ครั้งแรก
		if err := s.Repo.CreateIdVerification(verify); err != nil {
			return errors.New("failed to create id verification")
		}
	} else {
		// ejected → update
		verify.ID = existing.ID
		if err := s.Repo.UpdateIdVerification(verify); err != nil {
			return errors.New("failed to update id verification")
		}
	}

	// consent (log ใหม่ได้)
	consents := []*domain.UserConsent{
		{
			UserID:      userID,
			ConsentCode: domain.ConsentDeclareTruth,
			Accepted:    true,
			AcceptedAt:  time.Now(),
		},
	}

	return s.Repo.CreateConsents(consents)
}
