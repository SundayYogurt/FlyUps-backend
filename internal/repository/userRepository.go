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
	FindUserByVerificationToken(token string) (*domain.User, error)
	FindUserByResetToken(token string) (*domain.User, error)
	FindUserById(id uint) (*domain.User, error)
	UpdateUser(user *domain.User) error
	UpdateUserProfile(userID uint, firstName, lastName, phone string, address *string) error
	UpsertStudentProfileByUserID(profile *domain.StudentProfile) error
	CreateVerificationRequests(idCard *domain.IdCardVerification, studentCard *domain.StudentCardVerification, consent []*domain.UserConsent) error
	HasPendingVerification(userID uint, verifyType string) (bool, error)
	CreateBankAccount(bank *domain.BankAccount) error
	UpdateBankAccount(bank *domain.BankAccount) error
	FindBankByUserId(userID uint) ([]domain.BankAccount, error)
	FindBankById(id uint) (*domain.BankAccount, error)
	FindBankByAccountNumber(accountNumber string) (*domain.BankAccount, error)
}

type userRepository struct {
	db *gorm.DB
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

func (r *userRepository) HasPendingVerification(userID uint, verifyType string) (bool, error) {
	var count int64
	var err error

	if verifyType == "id_card" {
		err = r.db.Model(&domain.IdCardVerification{}).
			Where("user_id = ? AND status IN ?", userID, []string{"pending", "approved"}).
			Count(&count).Error
	} else if verifyType == "student_card" {
		err = r.db.Model(&domain.StudentCardVerification{}).
			Where("user_id = ? AND status IN ?", userID, []string{"pending", "approved"}).
			Count(&count).Error
	}

	return count > 0, err
}

func (r *userRepository) CreateVerificationRequests(idVerify *domain.IdCardVerification, studentVerify *domain.StudentCardVerification, consent []*domain.UserConsent) error {

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(idVerify).Error; err != nil {
			return err
		}

		if studentVerify != nil {
			if err := tx.Create(&studentVerify).Error; err != nil {
				return err
			}
		}

		if consent != nil {
			if err := tx.Create(&consent).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *userRepository) FindUserByVerificationToken(token string) (*domain.User, error) {
	var user domain.User

	err := r.db.Where("verification_token = ?", token).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateUser(user *domain.User) error {
	return r.db.Save(user).Error
}

func (r *userRepository) UpdateUserProfile(userID uint, firstName, lastName, phone string, address *string) error {
	updates := map[string]any{
		"first_name": firstName,
		"last_name":  lastName,
		"phone":      phone,
	}
	if address != nil {
		updates["address"] = address
	}

	return r.db.Model(&domain.User{}).Where("id = ?", userID).Updates(updates).Error
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
			"student_code":  profile.StudentCode,
			"faculty":       profile.Faculty,
			"major":         profile.Major,
			"bio":           profile.Bio,
			"portfolio":     profile.Portfolio,
			"skills":        profile.Skills,
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
		Preload("StudentProfile.University").
		Preload("BankAccount").
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
