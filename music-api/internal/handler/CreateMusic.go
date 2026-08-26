package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"music-api/internal/services"
)

func (h *MusicHandler) CreateMusic(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	artistName := strings.TrimSpace(r.FormValue("artist_name"))
	songTitle := strings.TrimSpace(r.FormValue("song_title"))
	genre := strings.TrimSpace(r.FormValue("genre"))
	audioKey := strings.TrimSpace(r.FormValue("audio_key"))

	// Validate required text fields before uploading anything.
	if artistName == "" {
		http.Error(w, services.ErrArtistNameRequired.Error(), http.StatusBadRequest)
		return
	}

	if songTitle == "" {
		http.Error(w, services.ErrSongTitleRequired.Error(), http.StatusBadRequest)
		return
	}

	if genre == "" {
		http.Error(w, services.ErrGenreRequired.Error(), http.StatusBadRequest)
		return
	}

	// Get image file.
	imageFile, imageHeader, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "image is required", http.StatusBadRequest)
		return
	}
	defer imageFile.Close()

	// Get audio file.
	audioFile, audioHeader, err := r.FormFile("audio")
	if err != nil {
		http.Error(w, "audio is required", http.StatusBadRequest)
		return
	}
	defer audioFile.Close()

	// Upload image to Cloudinary.
	imageResult, err := h.CloudinaryService.UploadImage(
		r.Context(),
		imageFile,
		imageHeader.Filename,
		"music/images",
	)
	if err != nil {
		http.Error(w, "failed to upload image", http.StatusInternalServerError)
		return
	}

	// Upload audio to Cloudinary.
	audioResult, err := h.CloudinaryService.UploadAudio(
		r.Context(),
		audioFile,
		audioHeader.Filename,
		"music/audio",
	)
	if err != nil {
		http.Error(w, "failed to upload audio", http.StatusInternalServerError)
		return
	}

	createdMusic, err := h.Service.CreateMusic(
		r.Context(),
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
		if isValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		http.Error(w, "failed to create music post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createdMusic); err != nil {
		return
	}
}

// isValidationError reports whether err is one of the service's field-level
// validation errors, which map to 400 Bad Request.
func isValidationError(err error) bool {
	return errors.Is(err, services.ErrArtistNameRequired) ||
		errors.Is(err, services.ErrSongTitleRequired) ||
		errors.Is(err, services.ErrGenreRequired)
}
