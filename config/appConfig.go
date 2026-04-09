package config

import (
	"errors"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	ServerPort          string
	Dsn                 string
	AppSecret           string
	ResendAPIKey        string
	EmailFrom           string
	BaseURL             string
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	StripeSecretKey     string
	StripeWebhookSecret string
	IAppAPIKey          string
	GoogleClientID      string
	GoogleClientSecret  string
	GoogleRedirectURI   string
}

func SetupEnv() (cfg AppConfig, err error) {

	// if os.Getenv("APP_ENV") == "dev" {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
	// }

	httpPort := os.Getenv("HTTP_PORT")
	if len(httpPort) < 1 {
		return AppConfig{}, errors.New("env variables not found")
	}

	Dsn := os.Getenv("DSN")
	if len(Dsn) < 1 {
		return AppConfig{}, errors.New("DSN env variables not found")
	}

	appSecret := os.Getenv("APP_SECRET")
	if len(appSecret) < 1 {
		return AppConfig{}, errors.New("appSecret env variables not found")
	}

	resendAPIKey := os.Getenv("RESEND_API_KEY")

	if len(resendAPIKey) < 1 {
		return AppConfig{}, errors.New("resendAPIKey env variables not found")
	}
	emailFrom := os.Getenv("EMAIL_FROM")

	if len(emailFrom) < 1 {
		return AppConfig{}, errors.New("emailFrom env variables not found")
	}

	baseURL := os.Getenv("BASE_URL")

	if len(baseURL) < 1 {
		return AppConfig{}, errors.New("baseURL env variables not found")
	}

	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	if len(cloudName) < 1 {
		return AppConfig{}, errors.New("cloudinary cloud name not found")
	}

	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	if len(apiKey) < 1 {
		return AppConfig{}, errors.New("cloudinary api key not found")
	}

	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")
	if len(apiSecret) < 1 {
		return AppConfig{}, errors.New("cloudinary api secret not found")
	}

	stripeSecretKey := os.Getenv("STRIPE_SECRET_KEY")
	stripeWebhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	iappAPIKey := os.Getenv("IAPP_API_KEY")
	if len(iappAPIKey) < 1 {
		return AppConfig{}, errors.New("iapp api key not found")
	}

	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	if len(googleClientID) < 1 {
		return AppConfig{}, errors.New("google client id env variables not found")
	}

	googleClientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	if len(googleClientSecret) < 1 {
		return AppConfig{}, errors.New("google client secret env variables not found")
	}

	googleRedirectURL := os.Getenv("GOOGLE_REDIRECT_URL")
	if len(googleRedirectURL) < 1 {
		return AppConfig{}, errors.New("google redirect url env variables not found")
	}

	return AppConfig{
		ServerPort:          httpPort,
		Dsn:                 Dsn,
		AppSecret:           appSecret,
		ResendAPIKey:        resendAPIKey,
		EmailFrom:           emailFrom,
		BaseURL:             baseURL,
		CloudinaryCloudName: cloudName,
		CloudinaryAPIKey:    apiKey,
		CloudinaryAPISecret: apiSecret,
		StripeSecretKey:     stripeSecretKey,
		StripeWebhookSecret: stripeWebhookSecret,
		IAppAPIKey:          iappAPIKey,
		GoogleClientID:      googleClientID,
		GoogleClientSecret:  googleClientSecret,
		GoogleRedirectURI:   googleRedirectURL,
	}, nil
}
