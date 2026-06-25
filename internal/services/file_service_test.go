package services

import (
	"bytes"
	"context"
	"errors"
	"flyup/internal/domain"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// testFile adapts *bytes.Reader to multipart.File
type testFile struct{ *bytes.Reader }

func (f *testFile) Close() error { return nil }

func newTestFile(content []byte) multipart.File {
	return &testFile{bytes.NewReader(content)}
}

// mockCloudinaryClient satisfies cloudinaryClient interface
type mockCloudinaryClient struct{ mock.Mock }

func (m *mockCloudinaryClient) UploadImage(ctx context.Context, file multipart.File) (string, error) {
	args := m.Called(ctx, file)
	return args.String(0), args.Error(1)
}

func (m *mockCloudinaryClient) UploadVideo(ctx context.Context, file multipart.File) (string, error) {
	args := m.Called(ctx, file)
	return args.String(0), args.Error(1)
}

func (m *mockCloudinaryClient) UploadRawFile(ctx context.Context, file multipart.File, ext string) (string, error) {
	args := m.Called(ctx, file, ext)
	return args.String(0), args.Error(1)
}

// magic bytes สำหรับแต่ละ file type
var (
	pngMagic     = append([]byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 504)...)
	pdfMagic     = append([]byte("%PDF-1.4 content here"), make([]byte, 491)...)
	xlsxMagic    = append([]byte{0x50, 0x4b, 0x03, 0x04}, make([]byte, 508)...)
	unknownMagic = make([]byte, 512)
)

// Image

func TestUploadService_Image_Success(t *testing.T) {
	cld := new(mockCloudinaryClient)
	svc := &uploadService{cld: cld}

	cld.On("UploadImage", mock.Anything, mock.Anything).Return("https://cdn.test/img.png", nil)

	result, err := svc.UploadFile(context.Background(), newTestFile(pngMagic), &multipart.FileHeader{Filename: "photo.png"})

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn.test/img.png", result.URL)
	assert.Equal(t, domain.MediaTypeImage, result.Type)
	cld.AssertExpectations(t)
}

func TestUploadService_Image_CloudinaryError(t *testing.T) {
	cld := new(mockCloudinaryClient)
	svc := &uploadService{cld: cld}

	cld.On("UploadImage", mock.Anything, mock.Anything).Return("", errors.New("cloudinary error"))

	result, err := svc.UploadFile(context.Background(), newTestFile(pngMagic), &multipart.FileHeader{Filename: "photo.png"})

	assert.Error(t, err)
	assert.Nil(t, result)
	cld.AssertExpectations(t)
}

// PDF

func TestUploadService_PDF_Success(t *testing.T) {
	cld := new(mockCloudinaryClient)
	svc := &uploadService{cld: cld}

	cld.On("UploadRawFile", mock.Anything, mock.Anything, "pdf").Return("https://cdn.test/doc.pdf", nil)

	result, err := svc.UploadFile(context.Background(), newTestFile(pdfMagic), &multipart.FileHeader{Filename: "document.pdf"})

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn.test/doc.pdf", result.URL)
	assert.Equal(t, domain.MediaTypeRaw, result.Type)
	cld.AssertExpectations(t)
}

// XLSX

func TestUploadService_XLSX_Success(t *testing.T) {
	cld := new(mockCloudinaryClient)
	svc := &uploadService{cld: cld}

	cld.On("UploadRawFile", mock.Anything, mock.Anything, "xlsx").Return("https://cdn.test/data.xlsx", nil)

	result, err := svc.UploadFile(context.Background(), newTestFile(xlsxMagic), &multipart.FileHeader{Filename: "data.xlsx"})

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn.test/data.xlsx", result.URL)
	assert.Equal(t, domain.MediaTypeRaw, result.Type)
	cld.AssertExpectations(t)
}

// Unsupported

func TestUploadService_UnsupportedType(t *testing.T) {
	cld := new(mockCloudinaryClient)
	svc := &uploadService{cld: cld}

	result, err := svc.UploadFile(context.Background(), newTestFile(unknownMagic), &multipart.FileHeader{Filename: "file.xyz"})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported file type")
	cld.AssertExpectations(t)
}

// PDF extension mismatch (PDF magic bytes but wrong extension)

func TestUploadService_PDFMimeMismatch(t *testing.T) {
	cld := new(mockCloudinaryClient)
	svc := &uploadService{cld: cld}

	// PDF magic แต่ใช้ extension .txt → ไม่ match เงื่อนไข (isPdfExt && isPdfMime)
	result, err := svc.UploadFile(context.Background(), newTestFile(pdfMagic), &multipart.FileHeader{Filename: "file.txt"})

	assert.Error(t, err)
	assert.Nil(t, result)
	cld.AssertExpectations(t)
}