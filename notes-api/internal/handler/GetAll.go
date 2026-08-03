package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func (h *NoteHandler) GetAllNotes(w http.ResponseWriter, r *http.Request) {
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	genre := strings.TrimSpace(r.URL.Query().Get("genre"))
	sortBy := strings.TrimSpace(r.URL.Query().Get("sort"))

	page := 1
	limit := 10

	if pageValue := r.URL.Query().Get("page"); pageValue != "" {
		parsedPage, err := strconv.Atoi(pageValue)
		if err != nil || parsedPage < 1 {
			http.Error(
				w,
				"page must be a positive number",
				http.StatusBadRequest,
			)
			return
		}

		page = parsedPage
	}

	if limitValue := r.URL.Query().Get("limit"); limitValue != "" {
		parsedLimit, err := strconv.Atoi(limitValue)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			http.Error(
				w,
				"limit must be between 1 and 100",
				http.StatusBadRequest,
			)
			return
		}

		limit = parsedLimit
	}

	notes, err := h.Repo.GetAll(
		r.Context(),
		search,
		genre,
		sortBy,
		page,
		limit,
	)
	if err != nil {
		http.Error(
			w,
			"failed to get music posts: "+err.Error(),
			http.StatusInternalServerError,
		)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(notes); err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
		return
	}

}
