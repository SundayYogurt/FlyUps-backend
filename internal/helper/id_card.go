package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

func NormalizeThaiIDNumber(raw string) (string, error) {
	var digits strings.Builder
	for _, ch := range raw {
		switch {
		case ch >= '0' && ch <= '9':
			digits.WriteRune(ch)
		case ch == '-' || ch == ' ':
		default:
			return "", errors.New("invalid ID card number")
		}
	}
	number := digits.String()
	if len(number) != 13 {
		return "", errors.New("invalid ID card number")
	}
	sum := 0
	for i := 0; i < 12; i++ {
		sum += int(number[i]-'0') * (13 - i)
	}
	if (11-sum%11)%10 != int(number[12]-'0') {
		return "", errors.New("invalid ID card number")
	}
	return number, nil
}

func IDCardFingerprint(number, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("id-card-verification:" + number))
	return hex.EncodeToString(mac.Sum(nil))
}
