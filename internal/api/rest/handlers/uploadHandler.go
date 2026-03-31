package handlers

import (
	"flyup/internal/api/rest"
	"flyup/internal/service"
	"mime/multipart"

	"github.com/gofiber/fiber/v3"
)

type UploadHandler struct {
	svc service.UploadService
}

func SetupUploadRoutes(rh *rest.RestHandler) {
	app := rh.App

	uploadSvc := service.NewUploadService(rh.Cloudinary)

	handler := UploadHandler{
		svc: uploadSvc,
	}

	app.Post("/upload", handler.UploadFile)

}

func (h *UploadHandler) UploadFile(ctx fiber.Ctx) error {
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return rest.BadRequestError(ctx, "no file uploaded")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	result, err := h.svc.UploadFile(ctx.Context(), file, fileHeader)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "upload success", fiber.Map{
		"url":  result.URL,
		"type": result.Type,
	})
}
