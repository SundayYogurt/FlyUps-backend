package handler

import (
	"flyup/internal/api/rest"
	"flyup/internal/helper"
	"flyup/internal/services"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type UploadHandler struct {
	svc services.UploadService
}

func NewUploadHandler(svc services.UploadService) *UploadHandler {
	return &UploadHandler{svc: svc}
}

func SetupUploadRoutes(rh *rest.RestHandler) {
	app := rh.App

	uploadSvc := services.NewUploadService(rh.Cloudinary)

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
// @Param files formData file false "Files to upload (repeat this key for multiple files)"
// @Success 200 {object} object "File uploaded URL"
// @Failure 400 {object} object "Invalid file"
// @Failure 500 {object} object "Internal Server Error"
// @Router /upload [post]
func (h *UploadHandler) UploadFile(ctx fiber.Ctx) error {
	form, err := ctx.MultipartForm()
	if err != nil || form == nil {
		return rest.BadRequestError(ctx, "invalid multipart form")
	}
	var headers []*multipart.FileHeader
	for key, files := range form.File {
		if key != "file" && key != "files" {
			return rest.BadRequestError(ctx, "unsupported file field")
		}
		headers = append(headers, files...)
	}
	if len(headers) == 0 || len(headers) > 10 {
		return rest.BadRequestError(ctx, "upload between 1 and 10 files, at most 5 of each type")
	}
	if len(form.File["file"]) > 1 || (len(form.File["file"]) > 0 && len(form.File["files"]) > 0) {
		return rest.BadRequestError(ctx, "use either file or files")
	}
	// Validate the whole batch before uploading any item.
	counts := make(map[string]int)
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return rest.BadRequestError(ctx, "cannot read file")
		}
		err = services.ValidateUpload(file, header)
		if err == nil {
			buffer := make([]byte, 512)
			n, _ := file.Read(buffer)
			kind := strings.SplitN(http.DetectContentType(buffer[:n]), "/", 2)[0]
			if kind != "image" && kind != "video" {
				kind = "raw"
			}
			counts[kind]++
			if counts[kind] > helper.MaxFilesPerType {
				err = helper.InvalidInput("at most 5 %s files are allowed", kind)
			}
		}
		_ = file.Close()
		if err != nil {
			return rest.BadRequestError(ctx, err.Error())
		}
	}
	// Support both:
	// - single file: key "file"
	// - multiple files: repeat key "files"
	if form, err := ctx.MultipartForm(); err == nil && form != nil && len(form.File["files"]) > 0 {
		fileHeaders := form.File["files"]
		items := make([]fiber.Map, 0, len(fileHeaders))

		for _, fh := range fileHeaders {
			f, err := fh.Open()
			if err != nil {
				return rest.InternalError(ctx, err)
			}
			result, err := h.svc.UploadFile(ctx.Context(), f, fh)
			_ = f.Close()
			if err != nil {
				return rest.InternalError(ctx, err)
			}
			items = append(items, fiber.Map{
				"url":      result.URL,
				"type":     result.Type,
				"filename": fh.Filename,
			})
		}

		return rest.SuccessResponse(ctx, "upload success", fiber.Map{
			"items": items,
		})
	}

	// fallback: single file
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		return rest.BadRequestError(ctx, "no file uploaded")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return rest.InternalError(ctx, err)
	}
	defer func(file multipart.File) { _ = file.Close() }(file)

	result, err := h.svc.UploadFile(ctx.Context(), file, fileHeader)
	if err != nil {
		return rest.InternalError(ctx, err)
	}

	return rest.SuccessResponse(ctx, "upload success", fiber.Map{
		"url":  result.URL,
		"type": result.Type,
	})
}
