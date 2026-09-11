package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"music-api/internal/services"
)

type SearchHandler struct {
	Service *services.SearchService
}

func NewSearchHandler(
	service *services.SearchService,
) *SearchHandler {
	return &SearchHandler{
		Service: service,
	}
}

func (h *SearchHandler) Search(
	w http.ResponseWriter,
	r *http.Request,
) {
	values :=
		r.URL.Query()

	query :=
		values.Get("q")

	searchType :=
		values.Get("type")

	sort :=
		values.Get("sort")

	genre :=
		values.Get("genre")

	page :=
		services.DefaultSearchPage

	limit :=
		services.DefaultSearchLimit

	// =========================
	// PAGE
	// =========================

	if rawPage :=
		values.Get("page"); rawPage != "" {

		parsedPage, err :=
			strconv.Atoi(
				rawPage,
			)

		if err != nil ||
			parsedPage <= 0 {

			writeSearchError(
				w,
				http.StatusBadRequest,
				"INVALID_SEARCH_PAGE",
				"page must be a positive integer",
			)
			return
		}

		page =
			parsedPage
	}

	// =========================
	// LIMIT
	// =========================

	if rawLimit :=
		values.Get("limit"); rawLimit != "" {

		parsedLimit, err :=
			strconv.Atoi(
				rawLimit,
			)

		if err != nil ||
			parsedLimit <= 0 {

			writeSearchError(
				w,
				http.StatusBadRequest,
				"INVALID_SEARCH_LIMIT",
				"limit must be a positive integer",
			)
			return
		}

		limit =
			parsedLimit
	}

	// =========================
	// SERVICE
	// =========================

	result, err :=
		h.Service.Search(
			r.Context(),
			services.SearchOptions{
				Query: query,

				Type: searchType,

				Sort: sort,

				Genre: genre,

				Page: page,

				Limit: limit,
			},
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrSearchQueryRequired,
		):
			writeSearchError(
				w,
				http.StatusBadRequest,
				"SEARCH_QUERY_REQUIRED",
				"search query is required",
			)

		case errors.Is(
			err,
			services.ErrSearchQueryTooShort,
		):
			writeSearchError(
				w,
				http.StatusBadRequest,
				"SEARCH_QUERY_TOO_SHORT",
				"search query must contain at least 2 characters",
			)

		case errors.Is(
			err,
			services.ErrSearchQueryTooLong,
		):
			writeSearchError(
				w,
				http.StatusBadRequest,
				"SEARCH_QUERY_TOO_LONG",
				"search query must not exceed 100 characters",
			)

		case errors.Is(
			err,
			services.ErrInvalidSearchType,
		):
			writeSearchError(
				w,
				http.StatusBadRequest,
				"INVALID_SEARCH_TYPE",
				"type must be all, track, artist, or release",
			)

		case errors.Is(
			err,
			services.ErrInvalidSearchSort,
		):
			writeSearchError(
				w,
				http.StatusBadRequest,
				"INVALID_SEARCH_SORT",
				"sort must be relevance, newest, or popular",
			)

		case errors.Is(
			err,
			services.ErrInvalidSearchPage,
		):
			writeSearchError(
				w,
				http.StatusBadRequest,
				"INVALID_SEARCH_PAGE",
				"page must be a positive integer",
			)

		case errors.Is(
			err,
			services.ErrInvalidSearchLimit,
		):
			writeSearchError(
				w,
				http.StatusBadRequest,
				"INVALID_SEARCH_LIMIT",
				"limit must be between 1 and 50",
			)

		case errors.Is(
			err,
			services.ErrSearchGenreTooLong,
		):
			writeSearchError(
				w,
				http.StatusBadRequest,
				"SEARCH_GENRE_TOO_LONG",
				"genre must not exceed 50 characters",
			)

		default:
			log.Printf(
				"search failed: %v",
				err,
			)

			writeSearchError(
				w,
				http.StatusInternalServerError,
				"SEARCH_FAILED",
				"search could not be completed",
			)
		}

		return
	}

	// =========================
	// RESPONSE
	// =========================

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(
		http.StatusOK,
	)

	if err :=
		json.NewEncoder(w).Encode(
			result,
		); err != nil {

		log.Printf(
			"failed to encode search response: %v",
			err,
		)
	}
}

func writeSearchError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(
		status,
	)

	_ = json.NewEncoder(w).Encode(
		map[string]any{
			"error": map[string]string{
				"code": code,

				"message": message,
			},
		},
	)
}
