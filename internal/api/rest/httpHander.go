package rest

import (
	"flyup/config"
	"flyup/internal/helper"
	"flyup/internal/port/cache"
	"flyup/internal/service"
	"flyup/pkg/notification"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
)

type RestHandler struct {
	App           *fiber.App
	DB            *gorm.DB
	Auth          helper.Auth
	Middlewares   Middleware
	Config        config.AppConfig
	Validator     *validator.Validate
	Notification  notification.NotificationClient
	Cloudinary    *helper.CloudinaryService
	NotifSvc      service.NotificationService
	InvestmentSvc service.InvestmentService
	Cache         cache.Cache
	Chat          service.ChatService
}
