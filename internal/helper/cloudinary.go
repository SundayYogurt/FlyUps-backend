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

func (c *CloudinaryService) UploadImage(file multipart.File) (string, error) {

	ctx := context.Background()

	res, err := c.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder: "flyup/projects",
	})

	if err != nil {
		return "", err
	}

	return res.SecureURL, nil
}
