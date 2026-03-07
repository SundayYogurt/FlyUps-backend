package handlers

import (
	"flyup/internal/api/rest"
	_ "flyup/internal/dto"
	"flyup/internal/service"
	_ "net/http"

	"github.com/gofiber/fiber/v3"
	_ "github.com/gofiber/fiber/v3"
)

type UserHandler struct {
	svc service.UserService
}

func SetupUserRoutes(rh *rest.RestHandler) {
	app := rh.App

	// create an instance of user service & inject to handler
	svc := service.UserService{
		//Repo:   repository.NewUserRepository(rh.DB),
		//Auth:   rh.Auth,
		//Config: rh.Config,
	}

	handler := UserHandler{
		svc: svc,
	}

	pubRoutes := app.Group("/")
	pubRoutes.Post("/register", handler.Register)

}

func (h *UserHandler) Register(ctx fiber.Ctx) error {
	//user := dto.UserSignup{}
	//err := ctx.BodyParser(&user)
	//if err != nil {
	//	return ctx.Status(http.StatusBadRequest).JSON(&fiber.Map{
	//		"message": "please provide valid inputs",
	//	})
	//}
	//
	//token, err := h.svc.Signup(user)
	//if err != nil {
	//	return ctx.Status(http.StatusInternalServerError).JSON(&fiber.Map{
	//		"message": "error on signup",
	//	})
	//}
	//
	//return ctx.Status(http.StatusOK).JSON(&fiber.Map{
	//	"message": "register",
	//	"token":   token,
	//})
	return nil
}
