package helper

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryService struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinary(cloudName, apiKey, apiSecret string) (*CloudinaryService, error) {

	cld, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, err
	}

	return &CloudinaryService{cld: cld}, nil
}

const maxUploadFileSize int64 = 5 * 1024 * 1024

func validateFileSize(file multipart.File) error {
	current, err := file.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("get file position: %w", err)
	}

	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("get file size: %w", err)
	}

	// คืนตำแหน่งเดิมก่อนส่งไฟล์ไปอัปโหลด
	if _, err := file.Seek(current, io.SeekStart); err != nil {
		return fmt.Errorf("restore file position: %w", err)
	}

	if size > maxUploadFileSize {
		return fmt.Errorf("file must not exceed 5 MB")
	}

	return nil
}

func (c *CloudinaryService) UploadImage(ctx context.Context, file multipart.File) (string, error) {

	if err := validateFileSize(file); err != nil {
		return "", err
	}

	res, err := c.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder: "flyup/projects",
		Format: "webp",
	})

	if err != nil {
		return "", err
	}

	return res.SecureURL, nil
}

func (c *CloudinaryService) UploadVideo(
	ctx context.Context,
	file multipart.File,
) (string, error) {
	const maxVideoSize int64 = 50 * 1024 * 1024

	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return "", fmt.Errorf("get video size: %w", err)
	}

	// กลับไปต้นไฟล์ก่อนอัปโหลด
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("reset video position: %w", err)
	}

	if size > maxVideoSize {
		return "", fmt.Errorf("video must not exceed 50 MB")
	}

	res, err := c.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:       "flyup/projects/videos",
		ResourceType: "video",
	})
	if err != nil {
		return "", err
	}

	return res.SecureURL, nil
}

func (c *CloudinaryService) UploadRawFile(ctx context.Context, file multipart.File, ext string) (string, error) {

	if err := validateFileSize(file); err != nil {
		return "", err
	}

	res, err := c.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:       "flyup/projects/documents",
		ResourceType: "raw",
		Format:       ext,
	})

	if err != nil {
		return "", err
	}

	return res.SecureURL, nil
}
