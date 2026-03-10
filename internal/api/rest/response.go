package rest

import (
	"github.com/gofiber/fiber/v3"

	"net/http"
)

func ErrorMessage(ctx fiber.Ctx, status int, err error) error {
	return ctx.Status(status).JSON(fiber.Map{
		"message": err.Error(),
	})
}

func InternalError(ctx fiber.Ctx, err error) error {
	return ctx.Status(http.StatusInternalServerError).JSON(fiber.Map{
		"message": "internal server error",
	})
}

func BadRequestError(ctx fiber.Ctx, msg string) error {
	return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{
		"message": msg,
	})
}

func SuccessResponse(ctx fiber.Ctx, msg string, data interface{}) error {
	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"message": msg,
		"data":    data,
	})
}
