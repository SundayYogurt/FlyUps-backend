package helper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"flyup/internal/domain"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	CreateHashedPassword(string) (string, error)
	VerifyPassword(string, string) error
	GenerateToken(uint, string, string) (string, error)
	GenerateRefreshToken(uint, string, string) (string, error)
	VerifyToken(string) (domain.User, error)
	VerifyRefreshToken(string) (domain.User, error)
	GenerateCode() (string, error)
}

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

	//check ตัวเล็ก (a-z)
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
func (a Auth) VerifyPassword(password string, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// สร้าง token
func (a Auth) GenerateToken(id uint, email string, role string) (string, error) {
	//validate ข้อมูลเบื้องต้น
	if id == 0 || email == "" || role == "" {
		return "", errors.New("required inputs are missing to generate token")
	}

	// เข้ารหัส ID เพื่อซ่อน
	obfuscatedID, err := a.encryptID(id)
	if err != nil {
		return "", errors.New("failed to obfuscate id")
	}

	// สร้าง Claims
	claims := jwt.MapClaims{
		"sub":   obfuscatedID, // ใช้ sub แทน user_id เพื่อความเป็นมาตรฐานและซ่อนความหมาย
		"email": email,
		"role":  role,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(15 * time.Minute).Unix(), // access token: 15 นาที
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

		// ถอดรหัส ID
		if sub, ok := claims["sub"].(string); ok {
			decryptedID, err := a.decryptID(sub)
			if err != nil {
				return domain.User{}, errors.New("invalid subject in token")
			}
			user.ID = decryptedID
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

// GenerateRefreshToken สร้าง refresh token อายุ 30 วัน
func (a Auth) GenerateRefreshToken(id uint, email string, role string) (string, error) {
	if id == 0 || email == "" || role == "" {
		return "", errors.New("required inputs are missing")
	}
	obfuscatedID, err := a.encryptID(id)
	if err != nil {
		return "", errors.New("failed to obfuscate id")
	}
	claims := jwt.MapClaims{
		"sub":   obfuscatedID,
		"email": email,
		"role":  role,
		"type":  "refresh",
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(30 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.Secret))
}

// VerifyRefreshToken ตรวจสอบ refresh token และคืน User
func (a Auth) VerifyRefreshToken(tokenStr string) (domain.User, error) {
	if tokenStr == "" {
		return domain.User{}, errors.New("invalid token")
	}
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.Secret), nil
	})
	if err != nil {
		return domain.User{}, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return domain.User{}, errors.New("invalid token")
	}
	if claims["type"] != "refresh" {
		return domain.User{}, errors.New("not a refresh token")
	}
	user := domain.User{}
	if sub, ok := claims["sub"].(string); ok {
		decryptedID, err := a.decryptID(sub)
		if err != nil {
			return domain.User{}, errors.New("invalid subject in token")
		}
		user.ID = decryptedID
	}
	if email, ok := claims["email"].(string); ok {
		user.Email = email
	}
	if role, ok := claims["role"].(string); ok {
		user.Role = role
	}
	return user, nil
}

func (a Auth) GenerateCode() (string, error) {
	return GenerateRandomToken(32)
}

//func (a Auth) SendVerifyEmail(to string, token string) error {
//	return SendVerifyEmail(to, token, a.Config) // ส่ง config ที่เก็บไว้ใน struct ไป
//}

func (a Auth) GetCurrentUser(ctx fiber.Ctx) domain.User {
	local := ctx.Locals("user")
	if local == nil {
		return domain.User{}
	}
	// รองรับทั้ง domain.User และ *domain.User
	if u, ok := local.(domain.User); ok {
		return u
	}
	if u, ok := local.(*domain.User); ok && u != nil {
		return *u
	}
	return domain.User{}
}

// encryptID เข้ารหัส ID ให้เป็น string ที่อ่านไม่ออก
func (a Auth) encryptID(id uint) (string, error) {
	key := []byte(a.Secret)
	if len(key) < 16 {
		// ถ้า secret สั้นไป ให้เติม 0 ให้ครบ 16 bytes (AES-128)
		newKey := make([]byte, 16)
		copy(newKey, key)
		key = newKey
	} else if len(key) > 32 {
		key = key[:32]
	} else if len(key) > 16 && len(key) < 24 {
		newKey := make([]byte, 24)
		copy(newKey, key)
		key = newKey
	} else if len(key) > 24 && len(key) < 32 {
		newKey := make([]byte, 32)
		copy(newKey, key)
		key = newKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	idStr := strconv.FormatUint(uint64(id), 10)
	ciphertext := gcm.Seal(nonce, nonce, []byte(idStr), nil)

	return base64.URLEncoding.EncodeToString(ciphertext), nil
}

// decryptID ถอดรหัส ID กลับเป็น uint
func (a Auth) decryptID(encryptedStr string) (uint, error) {
	ciphertext, err := base64.URLEncoding.DecodeString(encryptedStr)
	if err != nil {
		return 0, err
	}

	key := []byte(a.Secret)
	if len(key) < 16 {
		newKey := make([]byte, 16)
		copy(newKey, key)
		key = newKey
	} else if len(key) > 32 {
		key = key[:32]
	} else if len(key) > 16 && len(key) < 24 {
		newKey := make([]byte, 24)
		copy(newKey, key)
		key = newKey
	} else if len(key) > 24 && len(key) < 32 {
		newKey := make([]byte, 32)
		copy(newKey, key)
		key = newKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return 0, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return 0, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return 0, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return 0, err
	}

	id64, err := strconv.ParseUint(string(plaintext), 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(id64), nil
}

var (
	nonDigit   = regexp.MustCompile(`[^\d+]`)
	thaiMobile = regexp.MustCompile(`^0[689]\d{8}$`)
)

// NormalizePhone แปลงเบอร์ทุกรูปแบบให้เหลือ 0XXXXXXXXX
func NormalizePhone(s string) string {
	s = nonDigit.ReplaceAllString(strings.TrimSpace(s), "")

	// +66812345678 → 0812345678
	s = strings.TrimPrefix(s, "+")
	if strings.HasPrefix(s, "66") && len(s) == 11 {
		s = "0" + s[2:]
	}
	return s
}

func IsValidThaiMobile(s string) bool {
	return thaiMobile.MatchString(NormalizePhone(s))
}

// NewValidator สร้าง validator.Validate ที่ลงทะเบียน custom rule ของโปรเจกต์ไว้แล้ว
// (เช่น "thaiphone" สำหรับตรวจว่าเป็นเบอร์มือถือไทยเท่านั้น) เพื่อให้ทุกจุดที่สร้าง
// validator ใหม่ใช้กติกาเดียวกัน
func NewValidator() *validator.Validate {
	v := validator.New()
	v.RegisterValidation("thaiphone", func(fl validator.FieldLevel) bool {
		return IsValidThaiMobile(fl.Field().String())
	})
	return v
}
