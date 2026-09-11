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
)

const (
	maxPodcastArtworkSize = int64(10 << 20)

	maxPodcastArtworkRequestSize = int64(12 << 20)

	maxPodcastEpisodeAudioSize = int64(200 << 20)

	maxPodcastEpisodeMediaRequestSize = int64(220 << 20)

	podcastMultipartMemory = int64(16 << 20)
)

type openedPodcastMediaFile struct {
	File multipart.File

	Header *multipart.FileHeader
}

type uploadedPodcastAsset struct {
	PublicID string

	ResourceType string
}

type podcastMediaTooLargeError struct {
	Field string

	MaxSize int64
}

func (e podcastMediaTooLargeError) Error() string {
	return fmt.Sprintf(
		"%s exceeds the maximum allowed size",
		e.Field,
	)
}

type podcastMediaTypeError struct {
	Field string

	ContentType string
}

func (e podcastMediaTypeError) Error() string {
	return fmt.Sprintf(
		"%s has unsupported media type %q",
		e.Field,
		e.ContentType,
	)
}

// UpdatePodcastArtwork replaces the artwork belonging to a
// podcast owned by the authenticated user.
//
// Cloudinary and PostgreSQL cannot participate in one atomic
// transaction, so the operation deliberately uses:
//
//  1. upload replacement
//  2. update database
//  3. clean replacement if database update fails
//  4. delete previous asset after database update succeeds
func (h *PodcastHandler) UpdatePodcastArtwork(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	podcastID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PODCAST_ID",
			"invalid podcast ID",
		)
		return
	}

	if h.CloudinaryService == nil {
		log.Printf(
			"UpdatePodcastArtwork: Cloudinary service is not configured",
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"podcast media service is unavailable",
		)
		return
	}

	previousDetails, err :=
		h.Service.GetOwnedPodcast(
			r.Context(),
			podcastID,
			userID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to load podcast",
		)
		return
	}

	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPodcastArtworkRequestSize,
		)

	if err := r.ParseMultipartForm(
		podcastMultipartMemory,
	); err != nil {
		writePodcastMultipartError(
			w,
			err,
			"podcast artwork upload is too large",
		)
		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	artwork, err :=
		openRequiredPodcastMediaFile(
			r.MultipartForm,
			"artwork",
			maxPodcastArtworkSize,
			allowedPodcastImageTypes(),
		)

	if err != nil {
		writePodcastMediaValidationError(
			w,
			err,
		)
		return
	}

	defer artwork.File.Close()

	uploaded, err :=
		h.CloudinaryService.UploadImage(
			r.Context(),
			artwork.File,
			artwork.Header.Filename,
			"podcasts/artwork",
		)

	if err != nil {
		log.Printf(
			"UpdatePodcastArtwork: Cloudinary upload failed: %v",
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"UPLOAD_FAILED",
			"failed to upload podcast artwork",
		)
		return
	}

	newAsset :=
		services.PodcastMediaAsset{
			URL: uploaded.URL,

			PublicID: uploaded.PublicID,
		}

	previous :=
		previousDetails.Podcast

	updated, err :=
		h.Service.UpdatePodcast(
			r.Context(),
			podcastID,
			userID,
			services.UpdatePodcastInput{
				Title: previous.Title,

				Description: previous.Description,

				Category: previous.Category,

				Artwork: &newAsset,

				Status: previous.Status,

				IsExplicit: previous.IsExplicit,
			},
		)

	if err != nil {
		cleanupPodcastAsset(
			h,
			r,
			uploaded.PublicID,
			"image",
			"UpdatePodcastArtwork: failed to clean new artwork",
		)

		writePodcastServiceError(
			w,
			err,
			"failed to update podcast artwork",
		)
		return
	}

	deletePreviousPodcastAsset(
		h,
		r,
		previous.ArtworkPublicID,
		uploaded.PublicID,
		"image",
	)

	WriteJSON(
		w,
		http.StatusOK,
		updated,
	)
}

// UpdateEpisodeMedia replaces an episode's audio, artwork,
// or both.
//
// At least one media field must be supplied.
func (h *PodcastHandler) UpdateEpisodeMedia(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	episodeID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_EPISODE_ID",
			"invalid podcast episode ID",
		)
		return
	}

	if h.CloudinaryService == nil {
		log.Printf(
			"UpdateEpisodeMedia: Cloudinary service is not configured",
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"podcast media service is unavailable",
		)
		return
	}

	previous, err :=
		h.Service.GetOwnedEpisode(
			r.Context(),
			episodeID,
			userID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to load podcast episode",
		)
		return
	}

	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPodcastEpisodeMediaRequestSize,
		)

	if err := r.ParseMultipartForm(
		podcastMultipartMemory,
	); err != nil {
		writePodcastMultipartError(
			w,
			err,
			"podcast episode media upload is too large",
		)
		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	audio, err :=
		openOptionalPodcastMediaFile(
			r.MultipartForm,
			"audio",
			maxPodcastEpisodeAudioSize,
			allowedPodcastAudioTypes(),
		)

	if err != nil {
		writePodcastMediaValidationError(
			w,
			err,
		)
		return
	}

	if audio != nil {
		defer audio.File.Close()
	}

	artwork, err :=
		openOptionalPodcastMediaFile(
			r.MultipartForm,
			"artwork",
			maxPodcastArtworkSize,
			allowedPodcastImageTypes(),
		)

	if err != nil {
		if audio != nil {
			_ = audio.File.Close()
		}

		writePodcastMediaValidationError(
			w,
			err,
		)
		return
	}

	if artwork != nil {
		defer artwork.File.Close()
	}

	if audio == nil &&
		artwork == nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"MEDIA_REQUIRED",
			"audio or artwork is required",
		)
		return
	}

	var newlyUploaded []uploadedPodcastAsset

	cleanupNewUploads := func() {
		for _, asset := range newlyUploaded {
			cleanupPodcastAsset(
				h,
				r,
				asset.PublicID,
				asset.ResourceType,
				"UpdateEpisodeMedia: failed to clean new Cloudinary asset",
			)
		}
	}

	var audioAsset *services.PodcastMediaAsset

	if audio != nil {
		result, err :=
			h.CloudinaryService.UploadAudio(
				r.Context(),
				audio.File,
				audio.Header.Filename,
				"podcasts/episodes/audio",
			)

		if err != nil {
			log.Printf(
				"UpdateEpisodeMedia: audio upload failed: %v",
				err,
			)

			WriteError(
				w,
				http.StatusInternalServerError,
				"UPLOAD_FAILED",
				"failed to upload podcast episode audio",
			)
			return
		}

		audioAsset =
			&services.PodcastMediaAsset{
				URL: result.URL,

				PublicID: result.PublicID,
			}

		newlyUploaded =
			append(
				newlyUploaded,
				uploadedPodcastAsset{
					PublicID: result.PublicID,

					// Cloudinary stores uploaded audio as
					// the video resource type.
					ResourceType: "video",
				},
			)
	}

	var artworkAsset *services.PodcastMediaAsset

	if artwork != nil {
		result, err :=
			h.CloudinaryService.UploadImage(
				r.Context(),
				artwork.File,
				artwork.Header.Filename,
				"podcasts/episodes/artwork",
			)

		if err != nil {
			cleanupNewUploads()

			log.Printf(
				"UpdateEpisodeMedia: artwork upload failed: %v",
				err,
			)

			WriteError(
				w,
				http.StatusInternalServerError,
				"UPLOAD_FAILED",
				"failed to upload podcast episode artwork",
			)
			return
		}

		artworkAsset =
			&services.PodcastMediaAsset{
				URL: result.URL,

				PublicID: result.PublicID,
			}

		newlyUploaded =
			append(
				newlyUploaded,
				uploadedPodcastAsset{
					PublicID: result.PublicID,

					ResourceType: "image",
				},
			)
	}

	updated, err :=
		h.Service.UpdateEpisode(
			r.Context(),
			episodeID,
			userID,
			services.UpdatePodcastEpisodeInput{
				Title: previous.Title,

				Description: previous.Description,

				SeasonNumber: previous.SeasonNumber,

				EpisodeNumber: previous.EpisodeNumber,

				EpisodeType: previous.EpisodeType,

				Audio: audioAsset,

				Artwork: artworkAsset,

				DurationMS: previous.DurationMS,

				Status: previous.Status,

				IsExplicit: previous.IsExplicit,

				ScheduledAt: previous.ScheduledAt,
			},
		)

	if err != nil {
		cleanupNewUploads()

		writePodcastServiceError(
			w,
			err,
			"failed to update podcast episode media",
		)
		return
	}

	if audioAsset != nil {
		deletePreviousPodcastAsset(
			h,
			r,
			previous.AudioPublicID,
			audioAsset.PublicID,
			"video",
		)
	}

	if artworkAsset != nil {
		deletePreviousPodcastAsset(
			h,
			r,
			previous.ArtworkPublicID,
			artworkAsset.PublicID,
			"image",
		)
	}

	WriteJSON(
		w,
		http.StatusOK,
		updated,
	)
}

func openRequiredPodcastMediaFile(
	form *multipart.Form,
	field string,
	maxSize int64,
	allowedTypes map[string]struct{},
) (*openedPodcastMediaFile, error) {
	file, err :=
		openOptionalPodcastMediaFile(
			form,
			field,
			maxSize,
			allowedTypes,
		)

	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil,
			fmt.Errorf(
				"%s is required",
				field,
			)
	}

	return file, nil
}

func openOptionalPodcastMediaFile(
	form *multipart.Form,
	field string,
	maxSize int64,
	allowedTypes map[string]struct{},
) (*openedPodcastMediaFile, error) {
	if form == nil {
		return nil, nil
	}

	headers :=
		form.File[field]

	if len(headers) == 0 {
		return nil, nil
	}

	if len(headers) > 1 {
		return nil,
			fmt.Errorf(
				"%s accepts only one file",
				field,
			)
	}

	header := headers[0]

	if header.Size <= 0 {
		return nil,
			fmt.Errorf(
				"%s is empty",
				field,
			)
	}

	if header.Size > maxSize {
		return nil,
			podcastMediaTooLargeError{
				Field: field,

				MaxSize: maxSize,
			}
	}

	file, err :=
		header.Open()

	if err != nil {
		return nil,
			fmt.Errorf(
				"failed to open %s",
				field,
			)
	}

	if err :=
		validatePodcastMediaType(
			file,
			field,
			allowedTypes,
		); err != nil {
		_ = file.Close()

		return nil, err
	}

	return &openedPodcastMediaFile{
		File: file,

		Header: header,
	}, nil
}

func validatePodcastMediaType(
	file multipart.File,
	field string,
	allowedTypes map[string]struct{},
) error {
	buffer :=
		make(
			[]byte,
			512,
		)

	n, err :=
		file.Read(
			buffer,
		)

	if err != nil &&
		!errors.Is(
			err,
			io.EOF,
		) {
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
		return podcastMediaTypeError{
			Field: field,

			ContentType: contentType,
		}
	}

	if _, err :=
		file.Seek(
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

func allowedPodcastImageTypes() map[string]struct{} {
	return map[string]struct{}{
		"image/jpeg": {},

		"image/png": {},

		"image/webp": {},
	}
}

func allowedPodcastAudioTypes() map[string]struct{} {
	return map[string]struct{}{
		"audio/mpeg": {},

		"audio/mp3": {},

		"audio/wav": {},

		"audio/wave": {},

		"audio/x-wav": {},

		"audio/ogg": {},

		"application/ogg": {},

		"audio/mp4": {},

		"audio/x-m4a": {},

		"audio/aac": {},

		"audio/webm": {},

		"video/webm": {},
	}
}

func writePodcastMultipartError(
	w http.ResponseWriter,
	err error,
	tooLargeMessage string,
) {
	var maxBytesErr *http.MaxBytesError

	if errors.As(
		err,
		&maxBytesErr,
	) {
		WriteError(
			w,
			http.StatusRequestEntityTooLarge,
			"REQUEST_TOO_LARGE",
			tooLargeMessage,
		)
		return
	}

	WriteError(
		w,
		http.StatusBadRequest,
		"INVALID_MULTIPART_FORM",
		"invalid multipart form",
	)
}

func writePodcastMediaValidationError(
	w http.ResponseWriter,
	err error,
) {
	var tooLarge podcastMediaTooLargeError

	if errors.As(
		err,
		&tooLarge,
	) {
		WriteError(
			w,
			http.StatusRequestEntityTooLarge,
			"FILE_TOO_LARGE",
			err.Error(),
		)
		return
	}

	var mediaType podcastMediaTypeError

	if errors.As(
		err,
		&mediaType,
	) {
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

func cleanupPodcastAsset(
	h *PodcastHandler,
	r *http.Request,
	publicID string,
	resourceType string,
	logMessage string,
) {
	publicID =
		strings.TrimSpace(
			publicID,
		)

	if publicID == "" ||
		h.CloudinaryService == nil {
		return
	}

	if err :=
		h.CloudinaryService.DeleteAsset(
			r.Context(),
			publicID,
			resourceType,
		); err != nil {
		log.Printf(
			"%s %q: %v",
			logMessage,
			publicID,
			err,
		)
	}
}

func deletePreviousPodcastAsset(
	h *PodcastHandler,
	r *http.Request,
	oldPublicID string,
	newPublicID string,
	resourceType string,
) {
	oldPublicID =
		strings.TrimSpace(
			oldPublicID,
		)

	newPublicID =
		strings.TrimSpace(
			newPublicID,
		)

	if oldPublicID == "" ||
		oldPublicID == newPublicID {
		return
	}

	cleanupPodcastAsset(
		h,
		r,
		oldPublicID,
		resourceType,
		"podcast media: failed to remove previous Cloudinary asset",
	)
}
