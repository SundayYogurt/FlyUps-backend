package rest

import (
	"flyup/internal/domain"
	"flyup/internal/helper"
	"flyup/internal/repository"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type Middleware struct {
	Auth     helper.Auth
	UserRepo repository.UserRepository
}

func SetupMiddleware(auth helper.Auth, userRepo repository.UserRepository) Middleware {
	return Middleware{
		Auth:     auth,
		UserRepo: userRepo,
	}
}

func (m Middleware) Authorize(ctx fiber.Ctx) error {
	token := m.extractToken(ctx)

	if token == "" {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization token missing",
		})
	}

	userToken, err := m.Auth.VerifyToken(token)
	if err != nil || userToken.ID == 0 {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization failed",
		})
	}

	// โหลด user จริงจาก DB
	user, err := m.UserRepo.FindUserById(userToken.ID)
	if err != nil {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "user not found",
		})
	}

	// กันคนโดนแบน
	if user.Status == domain.SUSPENDED {
		return ctx.Status(403).JSON(fiber.Map{
			"message": "your account has been suspended",
		})
	}

	ctx.Locals("user", user)

	return ctx.Next()
}

func (m Middleware) AuthorizePioneer(ctx fiber.Ctx) error {

	token := m.extractToken(ctx)

	if token == "" {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization token missing",
		})
	}

	user, err := m.Auth.VerifyToken(token)
	if err != nil {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization failed",
		})
	}

	if user.Role != "pioneer" {
		return ctx.Status(403).JSON(fiber.Map{
			"message": "access denied",
		})
	}

	if user.Status == domain.SUSPENDED {
		return ctx.Status(403).JSON(fiber.Map{
			"message": "your account has been suspended",
		})
	}

	ctx.Locals("user", user)

	return ctx.Next()
}

func (m Middleware) AuthorizePioneerAndBooster(ctx fiber.Ctx) error {

	token := m.extractToken(ctx)

	if token == "" {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization token missing",
		})
	}

	user, err := m.Auth.VerifyToken(token)
	if err != nil {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization failed",
		})
	}

	if user.Role != "pioneer" && user.Role != "booster" {
		return ctx.Status(403).JSON(fiber.Map{
			"message": "access denied",
		})
	}

	if user.Status == domain.SUSPENDED {
		return ctx.Status(403).JSON(fiber.Map{
			"message": "your account has been suspended",
		})
	}

	ctx.Locals("user", user)

	return ctx.Next()
}

func (m Middleware) AuthorizeAdmin(ctx fiber.Ctx) error {

	token := m.extractToken(ctx)

	if token == "" {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization token missing",
		})
	}

	user, err := m.Auth.VerifyToken(token)
	if err != nil {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization failed",
		})
	}

	if user.Role != "admin" {
		return ctx.Status(403).JSON(fiber.Map{
			"message": "access denied",
		})
	}

	ctx.Locals("user", user)

	return ctx.Next()
}

func (m Middleware) extractToken(ctx fiber.Ctx) string {

	authHeader := ctx.Get("Authorization")

	if authHeader != "" {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	return ctx.Cookies("auth_token")
}
