package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"music-api/internal/middleware"
	"music-api/internal/services"
)

func (h *MusicHandler) CreateMusic(w http.ResponseWriter, r *http.Request) {
	artistID, ok := middleware.ArtistIDFromContext(r.Context())
	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			WriteError(
				w,
				http.StatusRequestEntityTooLarge,
				"REQUEST_TOO_LARGE",
				"music upload is too large",
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

	artistName := strings.TrimSpace(r.FormValue("artist_name"))
	songTitle := strings.TrimSpace(r.FormValue("song_title"))
	genre := strings.TrimSpace(r.FormValue("genre"))
	audioKey := strings.TrimSpace(r.FormValue("audio_key"))

	if artistName == "" {
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			services.ErrArtistNameRequired.Error(),
		)
		return
	}

	if songTitle == "" {
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			services.ErrSongTitleRequired.Error(),
		)
		return
	}

	if genre == "" {
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			services.ErrGenreRequired.Error(),
		)
		return
	}

	imageFile, imageHeader, err := r.FormFile("image")
	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"IMAGE_REQUIRED",
			"image is required",
		)
		return
	}
	defer imageFile.Close()

	audioFile, audioHeader, err := r.FormFile("audio")
	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"AUDIO_REQUIRED",
			"audio is required",
		)
		return
	}
	defer audioFile.Close()

	imageResult, err := h.CloudinaryService.UploadImage(
		r.Context(),
		imageFile,
		imageHeader.Filename,
		"music/images",
	)
	if err != nil {
		log.Printf(
			"cloudinary image upload failed: %v",
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"UPLOAD_FAILED",
			"failed to upload image",
		)
		return
	}

	audioResult, err := h.CloudinaryService.UploadAudio(
		r.Context(),
		audioFile,
		audioHeader.Filename,
		"music/audio",
	)
	if err != nil {
		log.Printf(
			"cloudinary audio upload failed: %v",
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"UPLOAD_FAILED",
			"failed to upload audio",
		)
		return
	}

	createdMusic, err := h.Service.CreateMusic(
		r.Context(),
		artistID,
		services.CreateMusicInput{
			ArtistName:    artistName,
			SongTitle:     songTitle,
			Genre:         genre,
			ImageURL:      imageResult.URL,
			ImagePublicID: imageResult.PublicID,
			AudioURL:      audioResult.URL,
			AudioPublicID: audioResult.PublicID,
			AudioKey:      audioKey,
		},
	)
	if err != nil {
		switch {
		case isValidationError(err):
			WriteError(
				w,
				http.StatusBadRequest,
				"VALIDATION_ERROR",
				err.Error(),
			)

		case errors.Is(err, services.ErrUnauthorized):
			WriteError(
				w,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"artist authentication is required",
			)

		default:
			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to create music post",
			)
		}

		return
	}

	WriteJSON(w, http.StatusCreated, createdMusic)
}

func isValidationError(err error) bool {
	return errors.Is(err, services.ErrArtistNameRequired) ||
		errors.Is(err, services.ErrSongTitleRequired) ||
		errors.Is(err, services.ErrGenreRequired)
}
