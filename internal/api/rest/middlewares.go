package rest

import (
	"flyup/internal/helper"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type Middleware struct {
	Auth helper.Auth
}

func SetupMiddleware(auth helper.Auth) Middleware {
	return Middleware{
		Auth: auth,
	}
}

func (m Middleware) Authorize(ctx fiber.Ctx) error {

	token := m.extractToken(ctx)

	if token == "" {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization token missing",
		})
	}

	user, err := m.Auth.VerifyToken(token)
	if err != nil || user.ID == 0 {
		return ctx.Status(401).JSON(fiber.Map{
			"message": "authorization failed",
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
