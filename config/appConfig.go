package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

type AppConfig struct {
	ServerPort    string
	Dsn           string
	AppSecret     string
	EmailHost     string
	EmailPort     string
	EmailUser     string
	EmailPassword string
}

func SetupEnv() (cfg AppConfig, err error) {

	// if os.Getenv("APP_ENV") == "dev" {
	godotenv.Load()
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

	emailHost := os.Getenv("EMAIL_HOST")
	emailPort := os.Getenv("EMAIL_PORT")
	emailUser := os.Getenv("EMAIL_USER")
	emailPassword := os.Getenv("EMAIL_PASSWORD")
	if emailHost == "" || emailPort == "" || emailUser == "" || emailPassword == "" {
		return AppConfig{}, errors.New("email env variables not found")
	}

	return AppConfig{
		ServerPort:    httpPort,
		Dsn:           Dsn,
		AppSecret:     appSecret,
		EmailHost:     emailHost,
		EmailPort:     emailPort,
		EmailUser:     emailUser,
		EmailPassword: emailPassword,
	}, nil

}
