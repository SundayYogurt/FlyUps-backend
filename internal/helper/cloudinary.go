package helper

import (
	"context"
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

func (c *CloudinaryService) UploadImage(ctx context.Context, file multipart.File) (string, error) {

	res, err := c.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder: "flyup/projects",
	})

	if err != nil {
		return "", err
	}

	return res.SecureURL, nil
}

func (c *CloudinaryService) UploadVideo(ctx context.Context, file multipart.File) (string, error) {

	res, err := c.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder:       "flyup/projects/videos",
		ResourceType: "video", // ต้องระบุว่าเป็น video
	})

	if err != nil {
		return "", err
	}

	return res.SecureURL, nil
}

func (c *CloudinaryService) UploadRawFile(ctx context.Context, file multipart.File, ext string) (string, error) {

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
