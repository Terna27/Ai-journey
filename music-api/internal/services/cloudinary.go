package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryUploadResult struct {
	URL      string
	PublicID string
}

type CloudinaryService struct {
	client *cloudinary.Cloudinary
}

func NewCloudinaryService() (*CloudinaryService, error) {
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("cloudinary environment variables are not configured")
	}

	cld, err := cloudinary.NewFromParams(
		cloudName,
		apiKey,
		apiSecret,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}

	return &CloudinaryService{
		client: cld,
	}, nil
}

func (s *CloudinaryService) UploadImage(
	ctx context.Context,
	file multipart.File,
	filename string,
	folder string,
) (CloudinaryUploadResult, error) {

	result, err := s.client.Upload.Upload(
		ctx,
		file,
		uploader.UploadParams{
			Folder:       folder,
			PublicID:     filename,
			ResourceType: "image",
		},
	)

	if err != nil {
		return CloudinaryUploadResult{}, fmt.Errorf(
			"failed to upload image to cloudinary: %w",
			err,
		)
	}

	return CloudinaryUploadResult{
		URL:      result.SecureURL,
		PublicID: result.PublicID,
	}, nil
}

func (s *CloudinaryService) UploadAudio(
	ctx context.Context,
	file multipart.File,
	filename string,
	folder string,
) (CloudinaryUploadResult, error) {

	result, err := s.client.Upload.Upload(
		ctx,
		file,
		uploader.UploadParams{
			Folder:       folder,
			PublicID:     filename,
			ResourceType: "video",
		},
	)

	if err != nil {
		return CloudinaryUploadResult{}, fmt.Errorf(
			"failed to upload audio to cloudinary: %w",
			err,
		)
	}

	return CloudinaryUploadResult{
		URL:      result.SecureURL,
		PublicID: result.PublicID,
	}, nil
}
