package service

import (
	"context"
	"encoding/json"
	"errors"
	"flyup/config"
	"flyup/internal/domain"
	"flyup/internal/dto"
	"flyup/internal/helper"
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
	Signing(email string, password string) (string, error)
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
	ApproveIdCard(userID uint, adminID uint) error
	ApproveStudentCard(userID uint, adminID uint) error
	RejectIdCard(userID uint, adminID uint) error
	RejectStudentCard(userID uint, adminID uint) error
	SuspendUser(adminID uint, userID uint, reason string) error
	RollbackActiveUser(userID uint) error
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
	notifSvc NotificationService
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
		return err
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("fail to hash password")
	}

	user.PasswordHash = string(hashedPassword)

	updates := map[string]interface{}{
		"password_hash": string(hashedPassword),
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
		return err
	}

	if user.Role != "pending" {
		return errors.New("user role is already selected")
	}

	if newRole == "pioneer" {
		parts := strings.Split(user.Email, "@")
		if len(parts) < 2 {
			return errors.New("invalid email format")
		}
		domainName := parts[1]

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
	}

	updates := map[string]interface{}{
		"role": newRole,
	}

	return s.Repo.UpdateUser(userID, updates)
}

func NewUserService(
	repo repository.UserRepository,
	urepo repository.UniversityRepository,
	auth helper.AuthService,
	cfg config.AppConfig,
	notifSvc NotificationService,
) UserService {
	return &userService{
		Repo:     repo,
		URepo:    urepo,
		Auth:     auth,
		Config:   cfg,
		notifSvc: notifSvc,
	}
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
		return err
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

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("fail to hash password")
	}

	user.PasswordHash = string(hashedPassword)

	updates := map[string]interface{}{
		"password_hash": string(hashedPassword),
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
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	u := domain.University{
		NameTH:   req.NameTH,
		NameEN:   req.NameEN,
		Province: req.Province,
	}

	if err := s.URepo.Create(&u); err != nil {
		return nil, err
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
		return nil, err
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
		return nil, err
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
		return nil, err
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
		return nil, err
	}

	// สร้าง domain ใหม่
	d := domain.UniversityDomain{
		UniversityID: id,
		Domain:       domainStr,
		IsActive:     true,
	}

	if err := s.URepo.CreateDomain(&d); err != nil {
		return nil, err
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
		return err
	}

	return s.URepo.DeleteDomain(id)
}

func (s *userService) RollbackActiveUser(userID uint) error {
	if userID == 0 {
		return errors.New("invalid userId")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return err
	}

	if user.Status == domain.ACTIVE {
		return errors.New("user is already suspended")
	}

	updates := map[string]interface{}{
		"status":         domain.ACTIVE,
		"suspend_reason": nil,
		"suspended_at":   nil,
		"suspended_by":   nil,
	}

	return s.Repo.UpdateUser(userID, updates)
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

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return err
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
		return err
	}

	student, err := s.Repo.FindIdCardStatus(userID)
	if err != nil {
		return err
	}

	if student == nil {
		return errors.New("no student verification found")
	}

	switch student.Status {
	case domain.VerifyStatusPending:

	case domain.VerifyStatusApproved:
		return errors.New("student is already verified")

	case domain.VerifyStatusRejected:
		return errors.New("verification was rejected, user must resubmit")
	}

	v := &domain.IdCardVerification{
		UserID:     userID,
		Status:     domain.VerifyStatusRejected,
		ReviewedBy: &adminID,
	}

	if err := s.Repo.UpdateIdCardVerification(v); err != nil {
		return err
	}

	if s.notifSvc != nil {
		s.notifSvc.CreateAndPush(userID, domain.NotifVerificationRejected,
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
		return err
	}

	student, err := s.Repo.FindStudentStatus(userID)
	if err != nil {
		return err
	}

	if student == nil {
		return errors.New("no student verification found")
	}

	switch student.Status {
	case domain.VerifyStatusPending:

	case domain.VerifyStatusApproved:
		return errors.New("student is already verified")

	case domain.VerifyStatusRejected:
		return errors.New("verification was rejected, user must resubmit")
	}

	v := &domain.StudentCardVerification{
		UserID:     userID,
		Status:     domain.VerifyStatusRejected,
		ReviewedBy: &adminID,
	}

	if err := s.Repo.UpdateStudentCardVerification(v); err != nil {
		return err
	}

	if s.notifSvc != nil {
		s.notifSvc.CreateAndPush(userID, domain.NotifVerificationRejected,
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
		return err
	}

	student, err := s.Repo.FindStudentStatus(userID)
	if err != nil {
		return err
	}

	if student == nil {
		return errors.New("no student verification found")
	}

	switch student.Status {
	case domain.VerifyStatusPending:

	case domain.VerifyStatusApproved:
		return errors.New("student is already verified")

	case domain.VerifyStatusRejected:
		return errors.New("verification was rejected, user must resubmit")
	}

	now := time.Now()

	v := &domain.StudentCardVerification{
		UserID:     userID,
		Status:     domain.VerifyStatusApproved,
		VerifiedAt: &now,
		ReviewedBy: &adminID,
	}

	if err := s.Repo.UpdateStudentCardVerification(v); err != nil {
		return err
	}

	if s.notifSvc != nil {
		s.notifSvc.CreateAndPush(userID, domain.NotifVerificationApproved,
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
		return err
	}

	student, err := s.Repo.FindIdCardStatus(userID)
	if err != nil {
		return err
	}

	if student == nil {
		return errors.New("no student verification found")
	}

	switch student.Status {
	case domain.VerifyStatusPending:

	case domain.VerifyStatusApproved:
		return errors.New("student is already verified")

	case domain.VerifyStatusRejected:
		return errors.New("verification was rejected, user must resubmit")
	}

	now := time.Now()

	v := &domain.IdCardVerification{
		UserID:     userID,
		Status:     domain.VerifyStatusApproved,
		VerifiedAt: &now,
		ReviewedBy: &adminID,
	}

	if err := s.Repo.UpdateIdCardVerification(v); err != nil {
		return err
	}

	if s.notifSvc != nil {
		s.notifSvc.CreateAndPush(userID, domain.NotifVerificationApproved,
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
		return nil, err
	}

	if len(userBank) == 0 {
		return nil, errors.New("bank not found")
	}

	return userBank, nil
}

func (s *userService) UpdateBankAccount(userID uint, bankID uint, input dto.BankRequest) error {
	if userID == 0 || bankID == 0 {
		return errors.New("invalid id")
	}
	// 1. หา bank
	bank, err := s.Repo.FindBankById(bankID)
	if err != nil {
		return err
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

	// 5. save
	return s.Repo.UpdateBankAccount(bank)
}

func (s *userService) AddBankAccount(userID uint, input dto.BankRequest) error {
	if userID == 0 {
		return errors.New("invalid user ID")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return err
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
		return err
	}

	if existing != nil {
		return errors.New("account number already exists")
	}

	bank := &domain.BankAccount{
		UserID:        userID,
		BankName:      *input.BankName,
		AccountName:   *input.AccountName,
		AccountNumber: *input.AccountNumber,
	}

	return s.Repo.CreateBankAccount(bank)

}

func (s *userService) SignUp(input dto.UserSignUp) (string, error) {
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

	newUser := &domain.User{
		Email:                      email,
		PasswordHash:               hPassword,
		FirstName:                  input.FirstName,
		LastName:                   input.LastName,
		Phone:                      input.Phone,
		Role:                       input.Role, // pioneer หรือ booster
		Status:                     domain.ACTIVE,
		VerificationToken:          &verifyToken,
		VerificationTokenExpiresAt: &expireAt,
	}

	// สร้างก้อน Consent จาก AcceptTerms ใน DTO
	consent := &domain.UserConsent{
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
		verifyLink := strings.TrimRight(s.Config.BaseURL, "/") + "/verify?token=" + token

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
			return "", errors.New("invalid or expired token")
		}
		return "", errors.New("internal server error")
	}

	// เช็ค verify แล้วหรือยัง
	if user.EmailVerifiedAt != nil {
		return "", errors.New("email already verified")
	}

	// เช็ค token หมดอายุ
	if user.VerificationTokenExpiresAt == nil ||
		time.Now().After(*user.VerificationTokenExpiresAt) {
		return "", errors.New("invalid or expired token")
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

func (s *userService) Signing(email string, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.Repo.FindUser(email)
	if err != nil {
		return "", errors.New("invalid email or password") // ไม่บอกว่า email ไม่มี เพื่อความปลอดภัย
	}

	// Check if email is verified
	if user.EmailVerifiedAt == nil {
		return "", errors.New("please verify your email first")
	}

	// Check if user is suspended
	if user.Status == domain.SUSPENDED {
		return "", errors.New("your account has been suspended")
	}

	// Check if this is a Google-only account
	if user.GoogleSub != nil && user.PasswordHash == "" {
		return "", errors.New("this email is registered with Google. Please use 'Continue with Google' to login")
	}

	// Verify password
	err = s.Auth.VerifyPassword(password, user.PasswordHash)
	if err != nil {
		return "", errors.New("invalid email or password") // ไม่บอกว่า password ผิด เพื่อความปลอดภัย
	}

	// Generate token
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

	updates := map[string]interface{}{
		"password_hash":          string(hashedPassword),
		"reset_token_hash":       nil,
		"reset_token_expires_at": nil,
	}

	return s.Repo.UpdateUser(user.ID, updates)
}

func (s *userService) GetProfile(userID uint) (*domain.User, error) {
	if userID == 0 {
		return nil, errors.New("invalid user id")
	}

	user, err := s.Repo.FindUserById(userID)
	if err != nil {
		return nil, err
	}

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
		parts := strings.Split(googleUser.Email, "@")
		if len(parts) < 2 {
			return "", errors.New("invalid email format")
		}
		domainName := parts[1]

		findDomain, err := s.URepo.GetUniversityByDomain(domainName)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", errors.New("sorry, this email domain is not registered as a university")
			}
			return "", errors.New("internal server error, try again later")
		}
		if !findDomain.IsActive {
			return "", errors.New("this university is not active")
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
			role = "booster" // Fallback Default
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
			return err
		}
	}
	if studentProfile != nil {
		if err := s.Repo.UpsertStudentProfileByUserID(studentProfile); err != nil {
			return err
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
		return err
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
			return err
		}
	} else {
		// rejected → update
		verify.ID = existing.ID
		if err := s.Repo.UpdateStudentVerification(verify); err != nil {
			return err
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
		return err
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
			return err
		}
	} else {
		// ejected → update
		verify.ID = existing.ID
		if err := s.Repo.UpdateIdVerification(verify); err != nil {
			return err
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
