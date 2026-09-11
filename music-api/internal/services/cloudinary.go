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
	return s.upload(
		ctx,
		file,
		filename,
		folder,
		"image",
		"image",
	)
}

func (s *CloudinaryService) UploadAudio(
	ctx context.Context,
	file multipart.File,
	filename string,
	folder string,
) (CloudinaryUploadResult, error) {
	return s.upload(
		ctx,
		file,
		filename,
		folder,
		"video",
		"audio",
	)
}

func (s *CloudinaryService) UploadVideo(
	ctx context.Context,
	file multipart.File,
	filename string,
	folder string,
) (CloudinaryUploadResult, error) {
	return s.upload(
		ctx,
		file,
		filename,
		folder,
		"video",
		"video",
	)
}

func (s *CloudinaryService) upload(
	ctx context.Context,
	file multipart.File,
	filename string,
	folder string,
	resourceType string,
	mediaType string,
) (CloudinaryUploadResult, error) {
	publicID, err := generateCloudinaryPublicID(filename)
	if err != nil {
		return CloudinaryUploadResult{}, fmt.Errorf(
			"failed to generate %s public ID: %w",
			mediaType,
			err,
		)
	}

	result, err := s.client.Upload.Upload(
		ctx,
		file,
		uploader.UploadParams{
			Folder:       folder,
			PublicID:     publicID,
			ResourceType: resourceType,
		},
	)
	if err != nil {
		return CloudinaryUploadResult{}, fmt.Errorf(
			"failed to upload %s to cloudinary: %w",
			mediaType,
			err,
		)
	}

	if err := validateCloudinaryResult(
		result,
		mediaType,
	); err != nil {
		return CloudinaryUploadResult{}, err
	}

	return CloudinaryUploadResult{
		URL:      result.SecureURL,
		PublicID: result.PublicID,
	}, nil
}

func (s *CloudinaryService) DeleteAsset(
	ctx context.Context,
	publicID string,
	resourceType string,
) error {
	publicID = strings.TrimSpace(publicID)
	resourceType = strings.TrimSpace(resourceType)

	if publicID == "" {
		return nil
	}

	if resourceType != "image" &&
		resourceType != "video" {
		return fmt.Errorf(
			"unsupported cloudinary resource type: %s",
			resourceType,
		)
	}

	invalidate := true

	result, err := s.client.Upload.Destroy(
		ctx,
		uploader.DestroyParams{
			PublicID:     publicID,
			ResourceType: resourceType,
			Type:         "upload",
			Invalidate:   &invalidate,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"failed to delete cloudinary asset: %w",
			err,
		)
	}

	if result == nil {
		return fmt.Errorf(
			"cloudinary returned an empty delete result",
		)
	}

	switch strings.ToLower(
		strings.TrimSpace(result.Result),
	) {
	case "ok", "not found":
		return nil

	default:
		return fmt.Errorf(
			"cloudinary failed to delete asset: %s",
			result.Result,
		)
	}
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

	if message := cloudinaryResponseError(
		result.Response,
	); message != "" {
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
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

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

	return strings.Trim(
		builder.String(),
		"-",
	)
}
