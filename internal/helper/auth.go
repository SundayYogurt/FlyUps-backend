package helper

import (
	"errors"
	"flyup/internal/domain"
	"fmt"
	"regexp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	Secret string
}

func SetupAuth(s string) Auth {
	return Auth{
		Secret: s,
	}
}

// hashed รหัสผ่าน
func (a Auth) CreateHashedPassword(p string) (string, error) {
	// check ความยาว มากกส่า 8 ตัว
	if len(p) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}

	//check ตัวใหญ่ (A-Z)
	upper := regexp.MustCompile(`[A-Z]`)
	if !upper.MatchString(p) {
		return "", errors.New("password must contain at least one uppercase letter")
	}

	//check ตัวใหญ่ (A-Z)
	lower := regexp.MustCompile(`[a-z]`)
	if !lower.MatchString(p) {
		return "", errors.New("password must contain at least one lowercase letter")
	}

	//check ตัวเลข
	digit := regexp.MustCompile(`[0-9]`)
	if !digit.MatchString(p) {
		return "", errors.New("password must contain at least one number")
	}

	//check อักขระพิเศษ
	special := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)
	if !special.MatchString(p) {
		return "", errors.New("password must contain at least one special character")
	}

	hashP, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost) // ส่ง password เป็น []byte แล้ว default = 10
	if err != nil {
		return "", err
	}

	return string(hashP), nil
}

// check password (login)
func (a Auth) VerifyPassword(hashedPassword string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// สร้าง token
func (a Auth) GenerateToken(id uint, email string, role string) (string, error) {
	//validate ข้อมูลเบื้องต้น
	if id == 0 || email == "" || role == "" {
		return "", errors.New("required inputs are missing to generate token")
	}

	// สร้าง Claims
	claims := jwt.MapClaims{
		"user_id": id,
		"email":   email,
		"role":    role,
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Minute * 30).Unix(), // exp 30 วัน
	}

	// สร้าง token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//sign Secret Key
	tokenStr, err := token.SignedString([]byte(a.Secret))
	if err != nil {
		return "", errors.New("failed to sign token")
	}

	return tokenStr, nil
}

func (a Auth) VerifyToken(tokenStr string) (domain.User, error) {
	// check ส่ามี token มาไหม
	if tokenStr == "" {
		return domain.User{}, errors.New("invalid token")
	}

	//Parse และ Verify Signature
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// เช็ค  algorithm ว่าเป็น HS256 จริงไหม
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.Secret), nil
	})

	//ตรวจสอบ Error เช่น หมดอายุ หรือ Token ปลอม
	if err != nil {
		return domain.User{}, err
	}

	// แกะ claims ออกมาเป็น User
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		user := domain.User{}

		// แปลง float64 จาก json เป็น อันที่ต้องการ
		if id, ok := claims["user_id"].(float64); ok {
			user.ID = uint(id)
		}
		if email, ok := claims["email"].(string); ok {
			user.Email = email
		}
		if role, ok := claims["role"].(string); ok {
			user.Role = role
		}

		return user, nil
	}

	return domain.User{}, errors.New("invalid token")
}

func (a Auth) GenerateCode() (string, error) {
	return GenerateRandomToken(32)
}

//func (a Auth) SendVerifyEmail(to string, token string) error {
//	return SendVerifyEmail(to, token, a.Config) // ส่ง config ที่เก็บไว้ใน struct ไป
//}
