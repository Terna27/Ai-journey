package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	cloudName := strings.TrimSpace(os.Getenv("CLOUDINARY_CLOUD_NAME"))
	apiKey := strings.TrimSpace(os.Getenv("CLOUDINARY_API_KEY"))
	apiSecret := strings.TrimSpace(os.Getenv("CLOUDINARY_API_SECRET"))

	if cloudName == "" || apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf(
			"cloudinary environment variables are not configured",
		)
	}

	cld, err := cloudinary.NewFromParams(
		cloudName,
		apiKey,
		apiSecret,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to initialize cloudinary: %w",
			err,
		)
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
	publicID, err := generateCloudinaryPublicID(filename)
	if err != nil {
		return CloudinaryUploadResult{}, fmt.Errorf(
			"failed to generate image public ID: %w",
			err,
		)
	}

	result, err := s.client.Upload.Upload(
		ctx,
		file,
		uploader.UploadParams{
			Folder:       folder,
			PublicID:     publicID,
			ResourceType: "image",
		},
	)
	if err != nil {
		return CloudinaryUploadResult{}, fmt.Errorf(
			"failed to upload image to cloudinary: %w",
			err,
		)
	}

	if err := validateCloudinaryResult(result, "image"); err != nil {
		return CloudinaryUploadResult{}, err
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
	publicID, err := generateCloudinaryPublicID(filename)
	if err != nil {
		return CloudinaryUploadResult{}, fmt.Errorf(
			"failed to generate audio public ID: %w",
			err,
		)
	}

	result, err := s.client.Upload.Upload(
		ctx,
		file,
		uploader.UploadParams{
			Folder:       folder,
			PublicID:     publicID,
			ResourceType: "video",
		},
	)
	if err != nil {
		return CloudinaryUploadResult{}, fmt.Errorf(
			"failed to upload audio to cloudinary: %w",
			err,
		)
	}

	if err := validateCloudinaryResult(result, "audio"); err != nil {
		return CloudinaryUploadResult{}, err
	}

	return CloudinaryUploadResult{
		URL:      result.SecureURL,
		PublicID: result.PublicID,
	}, nil
}

func validateCloudinaryResult(
	result *uploader.UploadResult,
	mediaType string,
) error {
	if result == nil {
		return fmt.Errorf(
			"cloudinary returned an empty %s upload result",
			mediaType,
		)
	}

	if message := cloudinaryResponseError(result.Response); message != "" {
		return fmt.Errorf(
			"cloudinary %s upload failed: %s",
			mediaType,
			message,
		)
	}

	if strings.TrimSpace(result.SecureURL) == "" {
		return fmt.Errorf(
			"cloudinary returned an incomplete %s upload result: secure URL is missing",
			mediaType,
		)
	}

	if strings.TrimSpace(result.PublicID) == "" {
		return fmt.Errorf(
			"cloudinary returned an incomplete %s upload result: public ID is missing",
			mediaType,
		)
	}

	return nil
}

func cloudinaryResponseError(response interface{}) string {
	var data map[string]interface{}

	switch value := response.(type) {
	case *map[string]interface{}:
		if value == nil {
			return ""
		}

		data = *value

	case map[string]interface{}:
		data = value

	default:
		return ""
	}

	rawError, exists := data["error"]
	if !exists || rawError == nil {
		return ""
	}

	switch value := rawError.(type) {
	case map[string]interface{}:
		if message, ok := value["message"].(string); ok {
			return strings.TrimSpace(message)
		}

	case string:
		return strings.TrimSpace(value)
	}

	return "cloudinary returned an upload error"
}

func generateCloudinaryPublicID(filename string) (string, error) {
	name := filepath.Base(strings.TrimSpace(filename))

	extension := filepath.Ext(name)
	name = strings.TrimSuffix(name, extension)

	name = sanitizePublicID(name)

	if name == "" {
		name = "asset"
	}

	randomBytes := make([]byte, 6)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf(
			"failed to generate random identifier: %w",
			err,
		)
	}

	randomID := hex.EncodeToString(randomBytes)

	return fmt.Sprintf(
		"%s-%d-%s",
		name,
		time.Now().UnixMilli(),
		randomID,
	), nil
}

func sanitizePublicID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))

	var builder strings.Builder
	previousDash := false

	for _, char := range value {
		isLetter := char >= 'a' && char <= 'z'
		isNumber := char >= '0' && char <= '9'

		if isLetter || isNumber {
			builder.WriteRune(char)
			previousDash = false
			continue
		}

		if !previousDash && builder.Len() > 0 {
			builder.WriteByte('-')
			previousDash = true
		}
	}

	return strings.Trim(builder.String(), "-")
}
