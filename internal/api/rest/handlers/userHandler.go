package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/dto"
	"log"
	"net/http"
	"strings"

	"flyup/internal/repository"
	"flyup/internal/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"
)

type UserHandler struct {
	svc       service.UserService
	validator *validator.Validate
}

func SetupUserRoutes(rh *rest.RestHandler) {

	app := rh.App

	// create an instance of user service & inject to handler
	svc := service.UserService{
		Repo:   repository.NewUserRepository(rh.DB),
		URepo:  repository.NewUniversityRepository(rh.DB),
		Auth:   rh.Auth,
		Config: rh.Config,
	}

	handler := UserHandler{
		svc:       svc,
		validator: validator.New(),
	}

	pubRoutes := app.Group("/")
	pubRoutes.Post("/signup", handler.Signup)
	pubRoutes.Get("/verify-email", handler.VerifyEmail)
	pubRoutes.Post("/signin", handler.Signin)

}

func (h *UserHandler) Signup(ctx fiber.Ctx) error {
	user := dto.UserSignup{}

	// Step 1: Bind JSON Body
	if err := ctx.Bind().Body(&user); err != nil {
		// ใช้ ErrorMessage เพื่อส่ง Error จากการ Bind กลับไปตรงๆ
		return rest.ErrorMessage(ctx, http.StatusBadRequest, err)
	}

	// Step 2: Validate Data
	if err := h.validator.Struct(user); err != nil {
		// ใช้ BadRequestError พร้อมบอกรายละเอียดการ Validate
		return rest.BadRequestError(ctx, "Validation failed: "+err.Error())
	}

	// Step 3: Call Service Logic
	msg, err := h.svc.Signup(user)
	if err != nil {
		errStr := err.Error()

		// 1. กรณีข้อมูลซ้ำ (409 Conflict)
		if strings.Contains(errStr, "already registered") {
			return rest.ErrorMessage(ctx, http.StatusConflict, err)
		}

		// 2. กรณี Business Logic ไม่ผ่าน (400 Bad Request)
		// เพิ่มเช็คคำว่า "record not found" หรือ "domain"
		if strings.Contains(errStr, "password") ||
			strings.Contains(errStr, "มหาวิทยาลัย") ||
			strings.Contains(errStr, "domain") ||
			strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "invalid email") {
			return rest.BadRequestError(ctx, errStr)
		}

		// กรณี Error อื่นๆ (500 Internal Error)
		log.Printf("[Signup Error]: %v", err)
		return rest.InternalError(ctx, err)
	}

	// Success Response (201 Created)
	return rest.SuccessResponse(ctx, msg, nil)
}

func (h *UserHandler) VerifyEmail(ctx fiber.Ctx) error {

	token := ctx.Query("token")

	if token == "" {
		return rest.BadRequestError(ctx, "token is required")
	}

	req := dto.VerifyEmailRequest{
		Token: token,
	}

	msg, err := h.svc.VerifyEmail(req)
	if err != nil {
		return rest.BadRequestError(ctx, err.Error())
	}

	return rest.SuccessResponse(ctx, msg, nil)
}

func (h *UserHandler) Signin(ctx fiber.Ctx) error {
	signinInput := dto.UserSignin{}
	err := ctx.Bind().Body(&signinInput)
	if err != nil {
		return ctx.Status(http.StatusBadRequest).JSON(fiber.Map{
			"message": "please provide valid inputs",
		})
	}
	token, err := h.svc.Signin(signinInput.Email, signinInput.Password)

	if err != nil {

		// แยก error verify email
		if err.Error() == "please verify your email first" {
			return ctx.Status(http.StatusForbidden).JSON(fiber.Map{
				"message": err.Error(),
			})
		}

		log.Println("signin error:", err)

		return ctx.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"message": "please provide correct user id password",
		})
	}
	// set cookie
	ctx.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   false, // false ถ้า localhost
		Path:     "/",
		MaxAge:   60 * 60 * 24, // 1 day
	})

	return ctx.Status(http.StatusOK).JSON(fiber.Map{
		"message": "login",
		"token":   token,
	})
}
