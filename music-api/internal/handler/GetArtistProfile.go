package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
)

func (h *MusicHandler) GetArtistProfile(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, err := strconv.Atoi(
		r.PathValue("id"),
	)
	if err != nil || artistID < 1 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_ARTIST_ID",
			"invalid artist ID",
		)
		return
	}

	artist, err := h.ArtistService.GetByID(
		r.Context(),
		artistID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(
				w,
				http.StatusNotFound,
				"ARTIST_NOT_FOUND",
				"artist not found",
			)
			return
		}

		log.Printf(
			"GetArtistProfile: failed to load artist %d: %v",
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

	tracks, err := h.Service.GetMusicByArtistID(
		r.Context(),
		artist.ID,
	)
	if err != nil {
		log.Printf(
			"GetArtistProfile: failed to load tracks for artist %d: %v",
			artist.ID,
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to load artist tracks",
		)
		return
	}

	response := models.ArtistProfile{
		Artist: models.PublicArtist{
			ID:        artist.ID,
			Name:      artist.Name,
			CreatedAt: artist.CreatedAt,
		},
		Tracks: tracks,
	}

	WriteJSON(
		w,
		http.StatusOK,
		response,
	)
}
