package helper

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"flyup/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func SetupGoogleOAuth(cfg config.AppConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURI,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func Sha256HmacHex(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
