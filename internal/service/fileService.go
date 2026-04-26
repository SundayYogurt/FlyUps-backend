package service

import (
	"context"
	"errors"
	"flyup/internal/domain"
	"flyup/internal/helper"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
)

type UploadResult struct {
	URL  string
	Type domain.MediaType
}

type UploadService interface {
	UploadFile(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (*UploadResult, error)
}

type uploadService struct {
	cld *helper.CloudinaryService
}

func NewUploadService(cld *helper.CloudinaryService) UploadService {
	return &uploadService{
		cld: cld,
	}
}

func (s *uploadService) UploadFile(ctx context.Context, file multipart.File, fileHeader *multipart.FileHeader) (*UploadResult, error) {

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return nil, err
	}

	if _, err = file.Seek(0, 0); err != nil {
		return nil, err
	}
	var (
		url       string
		mediaType domain.MediaType
	)

	contentType := http.DetectContentType(buffer[:n])
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	cleanExt := strings.TrimPrefix(ext, ".")
	isExcelExt := ext == ".xlsx" || ext == ".xls"

	isExcelMime :=
		contentType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
			contentType == "application/vnd.ms-excel" ||
			contentType == "application/zip"
	
	isPdfExt := ext == ".pdf"
	isPdfMime := contentType == "application/pdf"

	switch {
	case strings.HasPrefix(contentType, "image/"):
		url, err = s.cld.UploadImage(ctx, file)
		mediaType = domain.MediaTypeImage

	case strings.HasPrefix(contentType, "video/"):
		url, err = s.cld.UploadVideo(ctx, file)
		mediaType = domain.MediaTypeVideo

	case (isExcelExt && isExcelMime) || (isPdfExt && isPdfMime):
		url, err = s.cld.UploadRawFile(ctx, file, cleanExt)
		mediaType = domain.MediaTypeRaw

	default:
		return nil, errors.New("unsupported file type")
	}

	if err != nil {
		return nil, err
	}

	return &UploadResult{
		URL:  url,
		Type: mediaType,
	}, nil
}
