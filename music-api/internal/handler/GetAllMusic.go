package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

func (h *MusicHandler) GetAllMusic(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	genre := strings.TrimSpace(r.URL.Query().Get("genre"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))

	page := 1
	limit := 10

	if value := r.URL.Query().Get("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			WriteError(
				w,
				http.StatusBadRequest,
				"INVALID_PAGE",
				"page must be a positive number",
			)
			return
		}

		page = parsed
	}

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			WriteError(
				w,
				http.StatusBadRequest,
				"INVALID_LIMIT",
				"limit must be between 1 and 100",
			)
			return
		}

		limit = parsed
	}

	musicList, err := h.Service.ListMusic(
		r.Context(),
		search,
		genre,
		sortBy,
		page,
		limit,
	)
	if err != nil {
		log.Printf("GetAllMusic failed: %v", err)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to get music posts",
		)
		return
	}

	WriteJSON(w, http.StatusOK, musicList)
}
