package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/services"
)

type PlaylistHandler struct {
	Service *services.PlaylistService
}

func NewPlaylistHandler(
	service *services.PlaylistService,
) *PlaylistHandler {
	return &PlaylistHandler{
		Service: service,
	}
}

type createPlaylistRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

type updatePlaylistRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

func (h *PlaylistHandler) CreatePlaylist(
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

	var req createPlaylistRequest

	defer r.Body.Close()

	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesErr,
		) {
			WriteError(
				w,
				http.StatusRequestEntityTooLarge,
				"REQUEST_TOO_LARGE",
				"request body is too large",
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

	playlist, err :=
		h.Service.CreatePlaylist(
			r.Context(),
			userID,
			req.Name,
			req.Description,
			req.IsPublic,
		)

	if err != nil {
		h.writePlaylistError(
			w,
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusCreated,
		playlist,
	)
}

func (h *PlaylistHandler) GetMyPlaylists(
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

	playlists, err :=
		h.Service.GetUserPlaylists(
			r.Context(),
			userID,
		)

	if err != nil {
		h.writePlaylistError(
			w,
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		playlists,
	)
}

func (h *PlaylistHandler) GetPlaylist(
	w http.ResponseWriter,
	r *http.Request,
) {
	playlistID, err :=
		parsePlaylistID(r)

	if err != nil ||
		playlistID <= 0 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PLAYLIST_ID",
			"invalid playlist ID",
		)
		return
	}

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

	playlist, err :=
		h.Service.GetPlaylist(
			r.Context(),
			playlistID,
			userID,
		)

	if err != nil {
		h.writePlaylistError(
			w,
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		playlist,
	)
}

func (h *PlaylistHandler) UpdatePlaylist(
	w http.ResponseWriter,
	r *http.Request,
) {
	playlistID, err :=
		parsePlaylistID(r)

	if err != nil ||
		playlistID <= 0 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PLAYLIST_ID",
			"invalid playlist ID",
		)
		return
	}

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

	var req updatePlaylistRequest

	defer r.Body.Close()

	if err := json.NewDecoder(
		r.Body,
	).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(
			err,
			&maxBytesErr,
		) {
			WriteError(
				w,
				http.StatusRequestEntityTooLarge,
				"REQUEST_TOO_LARGE",
				"request body is too large",
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

	playlist, err :=
		h.Service.UpdatePlaylist(
			r.Context(),
			playlistID,
			userID,
			req.Name,
			req.Description,
			req.IsPublic,
		)

	if err != nil {
		h.writePlaylistError(
			w,
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		playlist,
	)
}

func (h *PlaylistHandler) DeletePlaylist(
	w http.ResponseWriter,
	r *http.Request,
) {
	playlistID, err :=
		parsePlaylistID(r)

	if err != nil ||
		playlistID <= 0 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PLAYLIST_ID",
			"invalid playlist ID",
		)
		return
	}

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

	err = h.Service.DeletePlaylist(
		r.Context(),
		playlistID,
		userID,
	)

	if err != nil {
		h.writePlaylistError(
			w,
			err,
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func (h *PlaylistHandler) AddTrack(
	w http.ResponseWriter,
	r *http.Request,
) {
	playlistID, err :=
		parsePlaylistID(r)

	if err != nil ||
		playlistID <= 0 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PLAYLIST_ID",
			"invalid playlist ID",
		)
		return
	}

	musicID, err :=
		parseMusicID(r)

	if err != nil ||
		musicID <= 0 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_MUSIC_ID",
			"invalid music ID",
		)
		return
	}

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

	err = h.Service.AddTrack(
		r.Context(),
		playlistID,
		userID,
		musicID,
	)

	if err != nil {
		h.writePlaylistError(
			w,
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusCreated,
		map[string]string{
			"message": "track added to playlist",
		},
	)
}

func (h *PlaylistHandler) RemoveTrack(
	w http.ResponseWriter,
	r *http.Request,
) {
	playlistID, err :=
		parsePlaylistID(r)

	if err != nil ||
		playlistID <= 0 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PLAYLIST_ID",
			"invalid playlist ID",
		)
		return
	}

	musicID, err :=
		parseMusicID(r)

	if err != nil ||
		musicID <= 0 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_MUSIC_ID",
			"invalid music ID",
		)
		return
	}

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

	err = h.Service.RemoveTrack(
		r.Context(),
		playlistID,
		userID,
		musicID,
	)

	if err != nil {
		h.writePlaylistError(
			w,
			err,
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func (
	h *PlaylistHandler,
) writePlaylistError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrUserAuthenticationRequired,
	):
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)

	case errors.Is(
		err,
		services.ErrPlaylistNameRequired,
	):
		WriteError(
			w,
			http.StatusBadRequest,
			"PLAYLIST_NAME_REQUIRED",
			"playlist name is required",
		)

	case errors.Is(
		err,
		services.ErrPlaylistNameTooLong,
	):
		WriteError(
			w,
			http.StatusBadRequest,
			"PLAYLIST_NAME_TOO_LONG",
			"playlist name must not exceed 120 characters",
		)

	case errors.Is(
		err,
		services.ErrPlaylistNotFound,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"PLAYLIST_NOT_FOUND",
			"playlist not found",
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

	case errors.Is(
		err,
		services.ErrPlaylistTrackExists,
	):
		WriteError(
			w,
			http.StatusConflict,
			"TRACK_ALREADY_IN_PLAYLIST",
			"music is already in this playlist",
		)

	case errors.Is(
		err,
		services.ErrPlaylistTrackMissing,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"TRACK_NOT_IN_PLAYLIST",
			"music is not in this playlist",
		)

	default:
		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"playlist operation failed",
		)
	}
}

func parsePlaylistID(
	r *http.Request,
) (int64, error) {
	return strconv.ParseInt(
		r.PathValue("id"),
		10,
		64,
	)
}

func parseMusicID(
	r *http.Request,
) (int, error) {
	return strconv.Atoi(
		r.PathValue("musicID"),
	)
}
