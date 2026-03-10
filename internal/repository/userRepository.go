package repository

import (
	"flyup/internal/domain"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(usr domain.User, consent domain.UserConsent) (*domain.User, error)
	FindUser(email string) (*domain.User, error)
	FindUserByVerificationToken(token string) (*domain.User, error)
	UpdateUser(user *domain.User) error
}

type userRepository struct {
	db *gorm.DB
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

func (r *userRepository) FindUser(email string) (*domain.User, error) {
	var user domain.User

	// ใช้คำสั่ง Where เพื่อหาอีเมล และ First เพื่อดึงมาแค่ record เดียว
	err := r.db.Where("email = ?", email).First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) CreateUser(usr domain.User, consent domain.UserConsent) (*domain.User, error) {
	// ใช้ Transaction เพื่อกันข้อมูลไม่ครบ เช่น สมัครแล้วเน็ตดับ
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// create user
		if err := tx.Create(&usr).Error; err != nil { //ใช้ tx แทน db เหมือนลองก่อน แล้ว create address ของ usr แล้ว return error ถ้ามี error แล้ว ก็ return ออกไปยกเลิก การ create
			return err
		}
		// เอา id ที่ได้มาใส่ consent
		consent.UserID = usr.ID

		// สร้าง consent
		if err := tx.Create(&consent).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &usr, nil
}

// 3. Constructor สำหรับตอนเรียกใช้งาน รับ db เข้ามา แล้วจับยัดใส่ไว้ใน struct เตรียมพร้อมใช้งาน
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		db: db, // เก็บ db ไว้ใช้ในฟังก์ชัน
	}
}
