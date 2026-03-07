package rest

import (
	"flyup/config"
	"flyup/internal/helper"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type RestHandler struct {
	App       *fiber.App
	DB        *gorm.DB
	Auth      helper.Auth
	Config    config.AppConfig
	Validator *validator.Validate
}
