package repository

import (
	"errors"
	"flyup/internal/domain"
	"log"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(usr *domain.User, consent *domain.UserConsent) (*domain.User, error)
	FindUser(email string) (*domain.User, error)
	FindUserByResetToken(token string) (*domain.User, error)
	FindUserById(id uint) (*domain.User, error)
	UpdateUser(userID uint, updates map[string]interface{}) error
	UpsertStudentProfileByUserID(profile *domain.StudentProfile) error
	CreateBankAccount(bank *domain.BankAccount) error
	UpdateBankAccount(bank *domain.BankAccount) error
	FindBankByUserId(userID uint) ([]domain.BankAccount, error)
	FindBankById(id uint) (*domain.BankAccount, error)
	FindBankByAccountNumber(accountNumber string) (*domain.BankAccount, error)
	UpdateIdCardVerification(v *domain.IdCardVerification) error
	UpdateStudentCardVerification(v *domain.StudentCardVerification) error
	FindStudentStatus(userID uint) (*domain.StudentCardVerification, error)
	FindIdCardStatus(userID uint) (*domain.IdCardVerification, error)
	CreateIdVerification(v *domain.IdCardVerification) error
	CreateConsents(consents []*domain.UserConsent) error
	UpdateIdVerification(v *domain.IdCardVerification) error
	FindLatestIdVerification(userID uint) (*domain.IdCardVerification, error)
	UpdateStudentVerification(v *domain.StudentCardVerification) error
	FindLatestStudentVerification(userID uint) (*domain.StudentCardVerification, error)
	CreateStudentVerification(v *domain.StudentCardVerification) error
	FindAllByRole(role string) ([]domain.User, error)
	FindStudentRequest(status string) ([]domain.StudentCardVerification, error)
	FindUserIDCardRequest(status string) ([]domain.IdCardVerification, error)
	FindAllUsers(page, limit int, role, status, search string) ([]domain.User, int64, error)
	FindUniversityByUserId(userID uint) (*domain.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func (r *userRepository) FindUniversityByUserId(userID uint) (*domain.User, error) {
	var user domain.User

	err := r.db.Preload("StudentProfile").Where("id = ?", userID).First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindAllUsers(page, limit int, role, status, search string) ([]domain.User, int64, error) {
	var users []domain.User
	var total int64

	query := r.db.Model(&domain.User{})

	if role != "" {
		query = query.Where("role = ?", role)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if search != "" {
		query = query.Where("email LIKE ? OR first_name LIKE ? OR last_name LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err = query.Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) FindUserIDCardRequest(status string) ([]domain.IdCardVerification, error) {
	var cardsID []domain.IdCardVerification
	err := r.db.Preload("User").Where("status = ?", status).Order("created_at DESC").Find(&cardsID).Error
	if err != nil {
		return nil, err
	}
	return cardsID, nil
}

func (r *userRepository) FindStudentRequest(status string) ([]domain.StudentCardVerification, error) {
	var students []domain.StudentCardVerification
	err := r.db.Preload("User").Where("status = ?", status).Order("created_at DESC").Find(&students).Error
	if err != nil {
		return nil, err
	}
	return students, nil
}

func (r *userRepository) UpdateStudentVerification(v *domain.StudentCardVerification) error {
	return r.db.Model(&domain.StudentCardVerification{}).
		Where("id = ?", v.ID).
		Updates(map[string]interface{}{
			"document":    v.Document,
			"status":      v.Status,
			"verified_at": v.VerifiedAt,
			"reviewed_by": v.ReviewedBy,
		}).Error
}

func (r *userRepository) FindLatestStudentVerification(userID uint) (*domain.StudentCardVerification, error) {
	var v domain.StudentCardVerification

	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&v).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &v, err
}

func (r *userRepository) CreateStudentVerification(v *domain.StudentCardVerification) error {
	return r.db.Create(v).Error
}

func (r *userRepository) FindIdCardStatus(userID uint) (*domain.IdCardVerification, error) {
	idCard := &domain.IdCardVerification{}
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").First(idCard).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return idCard, nil
}

func (r *userRepository) FindStudentStatus(userID uint) (*domain.StudentCardVerification, error) {
	student := &domain.StudentCardVerification{}
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").First(student).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return student, nil
}

func (r *userRepository) UpdateIdCardVerification(v *domain.IdCardVerification) error {
	return r.db.Model(&domain.IdCardVerification{}).
		Where("user_id = ?", v.UserID).
		Updates(map[string]interface{}{
			"status":      v.Status,
			"verified_at": v.VerifiedAt,
			"reviewed_by": v.ReviewedBy,
		}).Error
}

func (r *userRepository) UpdateStudentCardVerification(v *domain.StudentCardVerification) error {
	return r.db.Model(&domain.StudentCardVerification{}).
		Where("user_id = ?", v.UserID).
		Updates(map[string]interface{}{
			"status":      v.Status,
			"verified_at": v.VerifiedAt,
			"reviewed_by": v.ReviewedBy,
		}).Error
}

func (r *userRepository) FindBankByAccountNumber(accountNumber string) (*domain.BankAccount, error) {
	var account domain.BankAccount
	err := r.db.Where("account_number = ?", accountNumber).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // ยังไม่มี
		}
		return nil, err // error จริง
	}
	return &account, nil
}

func (r *userRepository) FindBankById(id uint) (*domain.BankAccount, error) {
	var account domain.BankAccount
	err := r.db.First(&account, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // ไม่เจอ = ไม่ใช่ error
		}
		return nil, err // error จริง เช่น DB ล่ม
	}

	return &account, nil
}

func (r *userRepository) CreateBankAccount(bank *domain.BankAccount) error {
	return r.db.Create(bank).Error
}

func (r *userRepository) FindBankByUserId(userID uint) ([]domain.BankAccount, error) {
	var banks []domain.BankAccount

	err := r.db.Where("user_id = ?", userID).Find(&banks).Error
	if err != nil {
		return nil, err
	}

	return banks, nil
}

func (r *userRepository) UpdateBankAccount(bank *domain.BankAccount) error {
	return r.db.Save(bank).Error
}

func (r *userRepository) CreateIdVerification(v *domain.IdCardVerification) error {
	return r.db.Create(v).Error
}

func (r *userRepository) CreateConsents(consents []*domain.UserConsent) error {
	return r.db.Create(&consents).Error
}

func (r *userRepository) UpdateIdVerification(v *domain.IdCardVerification) error {
	return r.db.Model(&domain.IdCardVerification{}).
		Where("id = ?", v.ID).
		Updates(map[string]interface{}{
			"document":    v.Document,
			"selfie_url":  v.SelfieURL,
			"status":      v.Status,
			"face_score":  v.FaceScore,
			"ocr_payload": v.OcrPayload,
			"verified_at": v.VerifiedAt,
		}).Error
}

func (r *userRepository) FindLatestIdVerification(userID uint) (*domain.IdCardVerification, error) {
	var v domain.IdCardVerification
	err := r.db.
		Where("user_id = ?", userID).
		Order("created_at DESC").
		First(&v).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &v, err
}
func (r *userRepository) UpdateUser(userID uint, updates map[string]interface{}) error {
	return r.db.Model(&domain.User{}).
		Where("id = ?", userID).
		Updates(updates).Error
}

func (r *userRepository) UpsertStudentProfileByUserID(profile *domain.StudentProfile) error {
	if profile == nil {
		return nil
	}

	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing domain.StudentProfile
		err := tx.Where("user_id = ?", profile.UserID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(profile).Error
		}

		updates := map[string]any{
			"university_id": profile.UniversityID,
		}
		if profile.StudentCode != nil {
			updates["student_code"] = profile.StudentCode
		}
		if profile.Faculty != nil {
			updates["faculty"] = profile.Faculty
		}
		if profile.Major != nil {
			updates["major"] = profile.Major
		}
		if profile.Bio != nil {
			updates["bio"] = profile.Bio
		}
		if profile.Portfolio != nil {
			updates["portfolio"] = profile.Portfolio
		}
		if profile.Skills != nil {
			updates["skills"] = profile.Skills
		}
		return tx.Model(&domain.StudentProfile{}).Where("user_id = ?", profile.UserID).Updates(updates).Error
	})
}

func (r *userRepository) FindUser(email string) (*domain.User, error) {
	var user domain.User

	// ใช้คำสั่ง Where เพื่อหาอีเมล และ First เพื่อดึงมาแค่ record เดียว
	err := r.db.Where("email = ?", email).First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) CreateUser(usr *domain.User, consent *domain.UserConsent) (*domain.User, error) {
	// ใช้ Transaction เพื่อกันข้อมูลไม่ครบ เช่น สมัครแล้วเน็ตดับ
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// create user
		if err := tx.Create(usr).Error; err != nil { //ใช้ tx แทน db เหมือนลองก่อน แล้ว create address ของ usr แล้ว return error ถ้ามี error แล้ว ก็ return ออกไปยกเลิก การ create
			return err
		}
		// เอา id ที่ได้มาใส่ consent
		consent.UserID = usr.ID

		// สร้าง consent
		if err := tx.Create(consent).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return usr, nil
}

func (r *userRepository) FindUserByResetToken(token string) (*domain.User, error) {
	user := &domain.User{}

	if err := r.db.Where("reset_token_hash = ?", token).First(user).Error; err != nil {
		log.Printf("find user by reset token error: %v", err)
		return nil, errors.New("failed to find user by reset token")
	}

	return user, nil
}

func (r *userRepository) FindUserById(id uint) (*domain.User, error) {
	var user domain.User
	err := r.db.
		Preload("StudentProfile").
		Preload("BankAccount").
		Preload("StudentCardVerification", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(1)
		}).
		Preload("IdCardVerification", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(1)
		}).
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// 3. Constructor สำหรับตอนเรียกใช้งาน รับ db เข้ามา แล้วจับยัดใส่ไว้ใน struct เตรียมพร้อมใช้งาน
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db, // เก็บ db ไว้ใช้ในฟังก์ชัน
	}
}

func (r *userRepository) FindAllByRole(role string) ([]domain.User, error) {
	var users []domain.User
	err := r.db.Where("role = ?", role).Find(&users).Error
	return users, err
}
