package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/models"
	"music-api/internal/services"
)

const maxLyricsRequestBytes int64 = 1024 * 1024

type LyricsService interface {
	GetLyrics(
		ctx context.Context,
		musicID int,
	) (models.TrackLyrics, error)

	UpsertLyrics(
		ctx context.Context,
		musicID int,
		artistID int,
		in services.UpsertLyricsInput,
	) (models.TrackLyrics, error)

	DeleteLyrics(
		ctx context.Context,
		musicID int,
		artistID int,
	) error
}

type LyricsHandler struct {
	Service LyricsService
}

func NewLyricsHandler(
	service LyricsService,
) *LyricsHandler {
	return &LyricsHandler{
		Service: service,
	}
}

type upsertLyricsRequest struct {
	PlainLyrics string `json:"plain_lyrics"`

	SyncedLines []models.LyricLine `json:"synced_lines"`
}

func (h *LyricsHandler) GetLyrics(
	w http.ResponseWriter,
	r *http.Request,
) {
	musicID, ok :=
		parseLyricsMusicID(
			w,
			r,
		)

	if !ok {
		return
	}

	lyrics, err :=
		h.Service.GetLyrics(
			r.Context(),
			musicID,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrLyricsNotFound,
		):
			WriteError(
				w,
				http.StatusNotFound,
				"LYRICS_NOT_FOUND",
				"lyrics not found",
			)

		default:
			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to load lyrics",
			)
		}

		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		lyrics,
	)
}

func (h *LyricsHandler) UpsertLyrics(
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

	musicID, ok :=
		parseLyricsMusicID(
			w,
			r,
		)

	if !ok {
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxLyricsRequestBytes,
	)

	defer r.Body.Close()

	var req upsertLyricsRequest

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	if err := decoder.Decode(
		&req,
	); err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesErr,
		) {
			WriteError(
				w,
				http.StatusRequestEntityTooLarge,
				"REQUEST_TOO_LARGE",
				"lyrics request is too large",
			)

			return
		}

		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)

		return
	}

	lyrics, err :=
		h.Service.UpsertLyrics(
			r.Context(),
			musicID,
			artistID,
			services.UpsertLyricsInput{
				PlainLyrics: req.PlainLyrics,

				SyncedLines: req.SyncedLines,
			},
		)

	if err != nil {
		switch {
		case services.IsLyricsValidationError(
			err,
		):
			WriteError(
				w,
				http.StatusBadRequest,
				"VALIDATION_ERROR",
				err.Error(),
			)

		case errors.Is(
			err,
			services.ErrUnauthorized,
		):
			WriteError(
				w,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"artist authentication is required",
			)

		case errors.Is(
			err,
			services.ErrMusicNotFound,
		):
			WriteError(
				w,
				http.StatusNotFound,
				"MUSIC_NOT_FOUND",
				"music post not found",
			)

		default:
			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to save lyrics",
			)
		}

		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		lyrics,
	)
}

func (h *LyricsHandler) DeleteLyrics(
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

	musicID, ok :=
		parseLyricsMusicID(
			w,
			r,
		)

	if !ok {
		return
	}

	err :=
		h.Service.DeleteLyrics(
			r.Context(),
			musicID,
			artistID,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrUnauthorized,
		):
			WriteError(
				w,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"artist authentication is required",
			)

		case errors.Is(
			err,
			services.ErrLyricsNotFound,
		):
			WriteError(
				w,
				http.StatusNotFound,
				"LYRICS_NOT_FOUND",
				"lyrics not found",
			)

		default:
			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to delete lyrics",
			)
		}

		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func parseLyricsMusicID(
	w http.ResponseWriter,
	r *http.Request,
) (int, bool) {
	musicID, err :=
		strconv.Atoi(
			r.PathValue("id"),
		)

	if err != nil ||
		musicID < 1 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_MUSIC_ID",
			"invalid music post ID",
		)

		return 0, false
	}

	return musicID, true
}
