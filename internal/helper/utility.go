package helper

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/smtp"
)

type EmailConfig struct {
	Host     string
	Port     string
	Email    string
	Password string
}

func SendVerifyEmail(to string, token string, config EmailConfig) error {
	// ตั้งค่าหัวข้อ
	subject := "Subject: Verify Your Email - FlyUp\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"

	//สร้าง link สำหรับกด verify
	verifyLink := fmt.Sprintf("http://localhost:5173/verify?token=%s", token)

	body := fmt.Sprintf(`
		<html>
			<body>
				<h1>Welcome to FlyUp!</h1>
				<p>Please click the link below to verify your account:</p>
				<a href="%s" style="padding: 10px 20px; background-color: blue; color: white; text-decoration: none;">Verify Email</a>
				<p>This link will expire in 24 hours.</p>
			</body>
		</html>
	`, verifyLink)

	msg := []byte(subject + mime + body)

	// ยืนยันตัวตนกับ SMTP Server
	auth := smtp.PlainAuth("", config.Email, config.Password, config.Host)

	address := fmt.Sprintf("%s:%s", config.Host, config.Port)
	err := smtp.SendMail(address, auth, config.Email, []string{to}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email: %v", err)
	}

	return nil
}

func GenerateRandomToken(length int) (string, error) {
	// สร้าง byte slice
	b := make([]byte, length/2) //หาร 2 เพราะตอนแปลงเป็น hex มันจะยาวสองเท่า

	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("could not generate random token: %v", err)
	}

	return hex.EncodeToString(b), nil
}
