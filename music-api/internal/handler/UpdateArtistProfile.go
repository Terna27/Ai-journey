package handler

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"

	"music-api/internal/middleware"
	"music-api/internal/services"

	"github.com/jackc/pgx/v5"
)

const (
	maxArtistProfileRequestSize = int64(80 << 20)
	maxArtistImageSize          = int64(10 << 20)
	maxArtistVideoSize          = int64(50 << 20)
)

func (h *MusicHandler) UpdateArtistProfile(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, ok :=
		middleware.ArtistIDFromContext(
			r.Context(),
		)
	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	previousArtist, err :=
		h.ArtistService.GetByID(
			r.Context(),
			artistID,
		)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(
				w,
				http.StatusNotFound,
				"ARTIST_NOT_FOUND",
				"artist profile not found",
			)
			return
		}

		log.Printf(
			"UpdateArtistProfile: failed to load artist %d: %v",
			artistID,
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to load artist profile",
		)
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxArtistProfileRequestSize,
	)

	if err := r.ParseMultipartForm(16 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			WriteError(
				w,
				http.StatusRequestEntityTooLarge,
				"REQUEST_TOO_LARGE",
				"artist profile upload is too large",
			)
			return
		}

		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_MULTIPART_FORM",
			"invalid multipart form",
		)
		return
	}

	var input services.UpdateArtistProfileInput

	if values, exists :=
		r.MultipartForm.Value["bio"]; exists {
		bio := ""

		if len(values) > 0 {
			bio = values[0]
		}

		input.Bio = &bio
	}

	var newlyUploaded []uploadedArtistAsset

	cleanupNewUploads := func() {
		for _, asset := range newlyUploaded {
			if err := h.CloudinaryService.DeleteAsset(
				r.Context(),
				asset.PublicID,
				asset.ResourceType,
			); err != nil {
				log.Printf(
					"UpdateArtistProfile: failed to clean up new Cloudinary asset %q: %v",
					asset.PublicID,
					err,
				)
			}
		}
	}

	profileImage, err := openOptionalArtistFile(
		r.MultipartForm,
		"profile_image",
		maxArtistImageSize,
		allowedArtistImageTypes(),
	)
	if err != nil {
		writeArtistMediaValidationError(
			w,
			err,
		)
		return
	}

	if profileImage != nil {
		defer profileImage.File.Close()

		result, err :=
			h.CloudinaryService.UploadImage(
				r.Context(),
				profileImage.File,
				profileImage.Header.Filename,
				"artists/profile-images",
			)
		if err != nil {
			log.Printf(
				"UpdateArtistProfile: profile image upload failed: %v",
				err,
			)

			WriteError(
				w,
				http.StatusInternalServerError,
				"UPLOAD_FAILED",
				"failed to upload profile image",
			)
			return
		}

		input.ProfileImage =
			&services.ArtistMediaAsset{
				URL:      result.URL,
				PublicID: result.PublicID,
			}

		newlyUploaded = append(
			newlyUploaded,
			uploadedArtistAsset{
				PublicID:     result.PublicID,
				ResourceType: "image",
			},
		)
	}

	heroVideo, err := openOptionalArtistFile(
		r.MultipartForm,
		"hero_video",
		maxArtistVideoSize,
		allowedArtistVideoTypes(),
	)
	if err != nil {
		cleanupNewUploads()
		writeArtistMediaValidationError(
			w,
			err,
		)
		return
	}

	if heroVideo != nil {
		defer heroVideo.File.Close()

		result, err :=
			h.CloudinaryService.UploadVideo(
				r.Context(),
				heroVideo.File,
				heroVideo.Header.Filename,
				"artists/hero-videos",
			)
		if err != nil {
			cleanupNewUploads()

			log.Printf(
				"UpdateArtistProfile: hero video upload failed: %v",
				err,
			)

			WriteError(
				w,
				http.StatusInternalServerError,
				"UPLOAD_FAILED",
				"failed to upload hero video",
			)
			return
		}

		input.HeroVideo =
			&services.ArtistMediaAsset{
				URL:      result.URL,
				PublicID: result.PublicID,
			}

		newlyUploaded = append(
			newlyUploaded,
			uploadedArtistAsset{
				PublicID:     result.PublicID,
				ResourceType: "video",
			},
		)
	}

	poster, err := openOptionalArtistFile(
		r.MultipartForm,
		"hero_video_poster",
		maxArtistImageSize,
		allowedArtistImageTypes(),
	)
	if err != nil {
		cleanupNewUploads()
		writeArtistMediaValidationError(
			w,
			err,
		)
		return
	}

	if poster != nil {
		defer poster.File.Close()

		result, err :=
			h.CloudinaryService.UploadImage(
				r.Context(),
				poster.File,
				poster.Header.Filename,
				"artists/hero-posters",
			)
		if err != nil {
			cleanupNewUploads()

			log.Printf(
				"UpdateArtistProfile: hero poster upload failed: %v",
				err,
			)

			WriteError(
				w,
				http.StatusInternalServerError,
				"UPLOAD_FAILED",
				"failed to upload hero video poster",
			)
			return
		}

		input.HeroVideoPoster =
			&services.ArtistMediaAsset{
				URL:      result.URL,
				PublicID: result.PublicID,
			}

		newlyUploaded = append(
			newlyUploaded,
			uploadedArtistAsset{
				PublicID:     result.PublicID,
				ResourceType: "image",
			},
		)
	}

	updatedArtist, err :=
		h.ArtistService.UpdateProfile(
			r.Context(),
			artistID,
			input,
		)
	if err != nil {
		cleanupNewUploads()

		switch {
		case errors.Is(
			err,
			services.ErrArtistBioTooLong,
		):
			WriteError(
				w,
				http.StatusBadRequest,
				"VALIDATION_ERROR",
				err.Error(),
			)

		case errors.Is(err, pgx.ErrNoRows):
			WriteError(
				w,
				http.StatusNotFound,
				"ARTIST_NOT_FOUND",
				"artist profile not found",
			)

		default:
			log.Printf(
				"UpdateArtistProfile: database update failed for artist %d: %v",
				artistID,
				err,
			)

			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to update artist profile",
			)
		}

		return
	}

	if input.ProfileImage != nil {
		deletePreviousArtistAsset(
			h,
			r,
			previousArtist.ProfileImagePublicID,
			input.ProfileImage.PublicID,
			"image",
		)
	}

	if input.HeroVideo != nil {
		deletePreviousArtistAsset(
			h,
			r,
			previousArtist.HeroVideoPublicID,
			input.HeroVideo.PublicID,
			"video",
		)
	}

	if input.HeroVideoPoster != nil {
		deletePreviousArtistAsset(
			h,
			r,
			previousArtist.HeroVideoPosterPublicID,
			input.HeroVideoPoster.PublicID,
			"image",
		)
	}

	WriteJSON(
		w,
		http.StatusOK,
		updatedArtist,
	)
}

type openedArtistFile struct {
	File   multipart.File
	Header *multipart.FileHeader
}

type uploadedArtistAsset struct {
	PublicID     string
	ResourceType string
}

func openOptionalArtistFile(
	form *multipart.Form,
	field string,
	maxSize int64,
	allowedTypes map[string]struct{},
) (*openedArtistFile, error) {
	headers := form.File[field]

	if len(headers) == 0 {
		return nil, nil
	}

	if len(headers) > 1 {
		return nil, fmt.Errorf(
			"%s accepts only one file",
			field,
		)
	}

	header := headers[0]

	if header.Size <= 0 {
		return nil, fmt.Errorf(
			"%s is empty",
			field,
		)
	}

	if header.Size > maxSize {
		return nil, artistMediaTooLargeError{
			Field:   field,
			MaxSize: maxSize,
		}
	}

	file, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to open %s",
			field,
		)
	}

	if err := validateArtistFileType(
		file,
		field,
		allowedTypes,
	); err != nil {
		file.Close()
		return nil, err
	}

	return &openedArtistFile{
		File:   file,
		Header: header,
	}, nil
}

func validateArtistFileType(
	file multipart.File,
	field string,
	allowedTypes map[string]struct{},
) error {
	buffer := make([]byte, 512)

	n, err := file.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf(
			"failed to inspect %s",
			field,
		)
	}

	if n == 0 {
		return fmt.Errorf(
			"%s is empty",
			field,
		)
	}

	contentType :=
		http.DetectContentType(
			buffer[:n],
		)

	if _, allowed :=
		allowedTypes[contentType]; !allowed {
		return artistMediaTypeError{
			Field:       field,
			ContentType: contentType,
		}
	}

	if _, err := file.Seek(
		0,
		io.SeekStart,
	); err != nil {
		return fmt.Errorf(
			"failed to reset %s",
			field,
		)
	}

	return nil
}

func allowedArtistImageTypes() map[string]struct{} {
	return map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/webp": {},
	}
}

func allowedArtistVideoTypes() map[string]struct{} {
	return map[string]struct{}{
		"video/mp4":       {},
		"video/webm":      {},
		"video/quicktime": {},
	}
}

type artistMediaTooLargeError struct {
	Field   string
	MaxSize int64
}

func (e artistMediaTooLargeError) Error() string {
	return fmt.Sprintf(
		"%s exceeds the maximum allowed size",
		e.Field,
	)
}

type artistMediaTypeError struct {
	Field       string
	ContentType string
}

func (e artistMediaTypeError) Error() string {
	return fmt.Sprintf(
		"%s has unsupported media type %q",
		e.Field,
		e.ContentType,
	)
}

func writeArtistMediaValidationError(
	w http.ResponseWriter,
	err error,
) {
	var tooLarge artistMediaTooLargeError

	if errors.As(err, &tooLarge) {
		WriteError(
			w,
			http.StatusRequestEntityTooLarge,
			"FILE_TOO_LARGE",
			err.Error(),
		)
		return
	}

	var mediaType artistMediaTypeError

	if errors.As(err, &mediaType) {
		WriteError(
			w,
			http.StatusBadRequest,
			"UNSUPPORTED_MEDIA_TYPE",
			err.Error(),
		)
		return
	}

	WriteError(
		w,
		http.StatusBadRequest,
		"INVALID_MEDIA",
		err.Error(),
	)
}

func deletePreviousArtistAsset(
	h *MusicHandler,
	r *http.Request,
	oldPublicID *string,
	newPublicID string,
	resourceType string,
) {
	if oldPublicID == nil {
		return
	}

	oldID := strings.TrimSpace(
		*oldPublicID,
	)

	if oldID == "" ||
		oldID == strings.TrimSpace(newPublicID) {
		return
	}

	if err := h.CloudinaryService.DeleteAsset(
		r.Context(),
		oldID,
		resourceType,
	); err != nil {
		// Database already references the replacement asset.
		// Cleanup failure must therefore not roll back a valid
		// artist profile update.
		log.Printf(
			"UpdateArtistProfile: failed to remove old Cloudinary asset %q: %v",
			oldID,
			err,
		)
	}
}
