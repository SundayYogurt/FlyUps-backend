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

	// try header
	authHeader := ctx.Get("Authorization")
	var token string

	if authHeader != "" {
		token = strings.Replace(authHeader, "Bearer ", "", 1)
	}

	// try cookie
	if token == "" {
		token = ctx.Cookies("auth_token")
	}

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

	authHeader := ctx.Get("Authorization")
	token := strings.Replace(authHeader, "Bearer ", "", 1)

	if token == "" {
		token = ctx.Cookies("auth_token")
	}

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
