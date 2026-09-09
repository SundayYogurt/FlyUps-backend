package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flyup/internal/domain"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

//func SendVerifyEmail(to string, token string, config EmailConfig) error {
//	// ตั้งค่าหัวข้อ
//	subject := "Subject: Verify Your Email - FlyUp\n"
//	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
//
//	//สร้าง link สำหรับกด verify
//	verifyLink := fmt.Sprintf("http://localhost:5173/verify?token=%s", token)
//
//	body := fmt.Sprintf(`
//<!DOCTYPE html>
//<html>
//<head>
//<meta charset="UTF-8">
//</head>
//<body style="margin:0; padding:0; background-color:#f4f6f8; font-family:Arial, Helvetica, sans-serif;">
//
//	<table width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f4f6f8; padding:40px 0;">
//		<tr>
//			<td align="center">
//
//				<table width="500" cellpadding="0" cellspacing="0" style="background:white; border-radius:10px; padding:40px; box-shadow:0 4px 10px rgba(0,0,0,0.05);">
//
//					<tr>
//						<td align="center">
//							<h1 style="margin:0; color:#1a1a1a;">🚀 Welcome to FlyUp</h1>
//						</td>
//					</tr>
//
//					<tr>
//						<td style="padding-top:20px; color:#555; font-size:16px; line-height:1.6;">
//							Thanks for joining FlyUp!
//							Please confirm your email address by clicking the button below.
//						</td>
//					</tr>
//
//					<tr>
//						<td align="center" style="padding:30px 0;">
//							<a href="%s"
//								style="
//									background-color:#2563eb;
//									color:white;
//									padding:14px 28px;
//									text-decoration:none;
//									border-radius:6px;
//									font-size:16px;
//									font-weight:bold;
//									display:inline-block;
//								">
//								Verify Email
//							</a>
//						</td>
//					</tr>
//
//					<tr>
//						<td style="color:#777; font-size:14px;">
//							This verification link will expire in <b>24 hours</b>.
//						</td>
//					</tr>
//
//					<tr>
//						<td style="padding-top:30px; color:#999; font-size:12px;">
//							If you did not create an account, you can safely ignore this email.
//						</td>
//					</tr>
//
//					<tr>
//						<td align="center" style="padding-top:30px; font-size:12px; color:#aaa;">
//							© 2026 FlyUp. All rights reserved.
//						</td>
//					</tr>
//
//				</table>
//
//			</td>
//		</tr>
//	</table>
//
//</body>
//</html>
//`, verifyLink)
//
//	msg := []byte(subject + mime + body)
//
//	// ยืนยันตัวตนกับ SMTP Server
//	auth := smtp.PlainAuth("", config.Email, config.Password, config.Host)
//
//	address := fmt.Sprintf("%s:%s", config.Host, config.Port)
//	err := smtp.SendMail(address, auth, config.Email, []string{to}, msg)
//	if err != nil {
//		return fmt.Errorf("failed to send email: %v", err)
//	}
//
//	return nil
//}

func GenerateRandomToken(length int) (string, error) {
	// สร้าง byte slice
	b := make([]byte, length/2) //หาร 2 เพราะตอนแปลงเป็น hex มันจะยาวสองเท่า

	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("could not generate random token: %v", err)
	}

	return hex.EncodeToString(b), nil
}

func Sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func GetMilestonePercent(phase int) (int, error) {
	switch phase {
	case 1:
		return 15, nil
	case 2:
		return 25, nil
	case 3:
		return 25, nil
	case 4:
		return 35, nil
	default:
		return 0, errors.New("invalid milestone phase")
	}
}

var stateTransitions = map[domain.ProjectState][]domain.ProjectState{
	domain.StateDraft: {
		domain.StatePendingReview,
		domain.StateCancelled,
	},
	domain.StatePendingReview: {
		domain.StateFunding,
		domain.StateCancelled,
	},
	domain.StateFunding: {
		domain.StateExecuting,
		domain.StateClosed,
		domain.StateCancelled,
		domain.StateSuspended,
		domain.StatePendingEditReview,
	},
	domain.StateExecuting: {
		domain.StateClosed,
		domain.StateCancelled,
		domain.StateSuspended,
		domain.StatePendingEditReview,
	},
	domain.StateClosed:    {},
	domain.StateCancelled: {},
	// pending_cancel สามารถ approve (→ cancelled) หรือ reject (→ previous state) โดย admin
	domain.StatePendingCancel: {
		domain.StateCancelled,
		domain.StateFunding,
		domain.StateExecuting,
	},
	// suspended สามารถ restore กลับได้โดย admin เท่านั้น
	domain.StateSuspended: {
		domain.StateFunding,
		domain.StateExecuting,
	},
	// pending_edit_review: admin approve (→ previous state) หรือ reject (→ previous state พร้อม revert ข้อมูล)
	domain.StatePendingEditReview: {
		domain.StateFunding,
		domain.StateExecuting,
	},
}

func IsValidStateTransition(oldState, newState domain.ProjectState) bool {
	allowed, ok := stateTransitions[oldState]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == newState {
			return true
		}
	}
	return false
}

func IsUniqueConstraintError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505" // unique_violation
	}
	return false
}

func GenerateProjectSlug(title string, id uint) string {
	slug := strings.ToLower(title)
	// แทนที่ space และ special chars
	re := regexp.MustCompile(`[^a-z0-9\x{0E00}-\x{0E7F}]+`)
	slug = re.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	// เพิ่ม id ต่อท้ายกันซ้ำ
	return fmt.Sprintf("%s-%d", slug, id)
}
