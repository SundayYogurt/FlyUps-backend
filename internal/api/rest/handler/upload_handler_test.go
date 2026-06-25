package handler

import (
	"bytes"
	"context"
	"flyup/internal/services"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockUploadService struct {
	mock.Mock
}

func (m *MockUploadService) UploadFile(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (*services.UploadResult, error) {
	args := m.Called(ctx, file, fileHeader)
	return args.Get(0).(*services.UploadResult), args.Error(1)
}

func setupUploadTest(t *testing.T) (*fiber.App, *MockUploadService, *UploadHandler) {
	app := fiber.New()
	mockService := new(MockUploadService)
	handler := NewUploadHandler(mockService)
	return app, mockService, handler
}

func TestUploadHandler_UploadFile(t *testing.T) {
	app, mockService, handler := setupUploadTest(t)
	app.Post("upload", handler.UploadFile)
	mockService.On("UploadFile", mock.Anything, mock.Anything, mock.Anything).
		Return(&services.UploadResult{
			URL:  "https://res.cloudinary.com/xxx/test.png",
			Type: "image",
		}, nil)
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.png")
	assert.NoError(t, err)
	part.Write([]byte("fake image content"))
	writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType()) // สำคัญสุด

	resp, err := app.Test(req, fiber.TestConfig{
		Timeout: -1, // หรือ 0 = no timeout
	})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	mockService.AssertExpectations(t)
}
