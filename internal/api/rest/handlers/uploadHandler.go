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

// UploadFile godoc
// @Summary Upload File
// @Description Upload media files (images, doc, pdf, etc.) to Cloudinary
// @Tags Upload
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "File to upload"
// @Success 200 {object} object "File uploaded URL"
// @Failure 400 {object} object "Invalid file"
// @Failure 500 {object} object "Internal Server Error"
// @Router /upload [post]
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
