package handler

import (
	"log"
	"net/http"

	"music-api/internal/services"
)

type DiscoveryHandler struct {
	Service *services.DiscoveryService
}

func NewDiscoveryHandler(
	service *services.DiscoveryService,
) *DiscoveryHandler {
	return &DiscoveryHandler{
		Service: service,
	}
}

func (h *DiscoveryHandler) HeroArtists(
	w http.ResponseWriter,
	r *http.Request,
) {
	result, err :=
		h.Service.HeroArtists(
			r.Context(),
		)

	if err != nil {
		log.Printf(
			"HeroArtists failed: %v",
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"HERO_ARTISTS_FAILED",
			"failed to load hero artists",
		)

		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		result,
	)
}

func (h *DiscoveryHandler) Discover(
	w http.ResponseWriter,
	r *http.Request,
) {
	result, err :=
		h.Service.Discover(
			r.Context(),
		)

	if err != nil {
		log.Printf(
			"Discover failed: %v",
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"DISCOVERY_FAILED",
			"failed to load discovery content",
		)

		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		result,
	)
}
